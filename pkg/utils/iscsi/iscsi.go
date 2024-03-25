package iscsi

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/QQGoblin/extrootfs/pkg/utils/log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	iscsilib "github.com/kubernetes-csi/csi-lib-iscsi/iscsi"
	"github.com/pkg/errors"
)

type Disk struct {
	Portals         []string
	IQN             string
	Lun             int32
	SessionSecret   iscsilib.Secrets
	DiscoverySecret iscsilib.Secrets
	DevicePath      string
}

type Session struct {
	Protocol                   string
	ID                         int32
	Portal                     string
	IQN                        string
	Name                       string
	ConnectionState            string
	SessionState               string
	InternalIscsidSessionState string
}

func New(userName, password string, iqn string, portals []string) *Disk {

	secrets := iscsilib.Secrets{
		SecretsType: "chap",
		UserName:    userName,
		Password:    password,
	}

	return &Disk{
		Portals:       portals,
		IQN:           iqn,
		Lun:           0, // 这里 lun id 固定为0
		SessionSecret: secrets,
		//DiscoverySecret: secrets, // 关闭 DiscoverySecret ，当前 iscsi-initiator-utils 和 iscci-lib 似乎有兼容性问题
	}
}

func (d *Disk) AttachDisk() error {
	c := iscsilib.Connector{
		TargetIqn:      d.IQN,
		TargetPortals:  d.Portals,
		SessionSecrets: d.SessionSecret,
		//DiscoverySecrets: d.DiscoverySecret,
		DoDiscovery:     false,
		DoCHAPDiscovery: true,
		RetryCount:      60,
		CheckInterval:   5,
	}

	devicePath, err := iscsilib.Connect(c)
	if err != nil {
		return err
	}

	if devicePath == "" {
		return fmt.Errorf("connect reported success, but no path returned")
	}
	d.DevicePath = devicePath
	log.DebugLogMsg("connect iscsi disk %s", devicePath)
	return nil

}

func (d *Disk) ReopenDisk() error {

	s, err := d.GetSession()
	if err != nil {
		return errors.Wrap(err, "reopen disk failed")
	}
	if len(s) > 0 {
		if err := d.DetachDisk(); err != nil {
			return errors.Wrap(err, "reopen disk, close old iscsi connect")
		}
	}
	return d.AttachDisk()
}

func (d *Disk) SetKernalConfig() error {

	// TODO: set device timeout
	// echo -1 > /sys/block/<device>/device/timeout
	return nil
}

func (d *Disk) DetachDisk() error {
	iscsilib.Disconnect(d.IQN, d.Portals)
	return nil
}

// CheckSessionState iscsi 连接不存在或者状态异常时返回 error
func (d *Disk) CheckSessionState() error {

	var err error
	// 获取设备的所有 session
	sessions, err := d.GetSession()
	if err != nil || len(sessions) == 0 {
		return errors.New(fmt.Sprintf("check session state failed: %v", err))
	}
	// 当前不考虑多路径的情况，只要 iqn 有一个是将健康状态的，就认为 iscsi session 是可以用的
	for _, s := range sessions {
		if s.SessionState == "LOGGED_IN" && s.ConnectionState == "LOGGED IN" {
			return nil
		}
	}
	return errors.New(fmt.Sprintf("check session state failed: %v", sessions))
}

// GetSession 返回iscsi连接信息
func (d *Disk) GetSession() ([]Session, error) {

	// 考虑可能存在多路径的场景，因此返回的是一个列表
	resp := make([]Session, 0)
	out, err := iscsilib.GetSessions()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok && exitErr.ProcessState.Sys().(syscall.WaitStatus).ExitStatus() == 21 {
			return resp, nil
		}
		return nil, err
	}

	sessions := parseSessions(out)
	for _, s := range sessions {
		if d.IQN == s.IQN {

			args := []string{"-m", "session", "--sid", strconv.Itoa(int(s.ID)), "-P", "3"}
			sinfo, err := iscsilib.ExecWithTimeout("iscsiadm", args, time.Second*3)
			if err != nil {
				resp = append(resp, s)
				continue
			}
			s.ConnectionState = regexpSessionState(sinfo, `iSCSI Connection State:(.*)`)
			s.SessionState = regexpSessionState(sinfo, `iSCSI Session State:(.*)`)
			s.InternalIscsidSessionState = regexpSessionState(sinfo, `Internal iscsid Session State:(.*)`)
			resp = append(resp, s)
		}
	}
	return resp, nil
}

// parseSession takes the raw stdout from the iscsiadm -m session command and encodes it into an iSCSI session type
func parseSessions(lines string) []Session {
	entries := strings.Split(strings.TrimSpace(lines), "\n")
	r := strings.NewReplacer("[", "",
		"]", "")

	var sessions []Session
	for _, entry := range entries {
		e := strings.Fields(entry)
		if len(e) < 4 {
			continue
		}
		protocol := strings.Split(e[0], ":")[0]
		id := r.Replace(e[1])
		id64, _ := strconv.ParseInt(id, 10, 32)
		portal := strings.Split(e[2], ",")[0]

		s := Session{
			Protocol: protocol,
			ID:       int32(id64),
			Portal:   portal,
			IQN:      e[3],
			Name:     strings.Split(e[3], ":")[1],
		}
		sessions = append(sessions, s)
	}
	return sessions
}

func regexpSessionState(sessionInfo []byte, reStr string) string {
	// TODO: 需要提供一个完整的接口用于获取所有信息
	regexpObj := regexp.MustCompile(reStr)

	s := regexpObj.FindStringSubmatch(string(sessionInfo))
	if len(s) <= 1 {
		return ""
	} else {
		return strings.TrimSpace(s[1])
	}

}

func (s *Session) String() string {
	return fmt.Sprintf("session name: %s,, status: Connection(%s), Session(%s)", s.Name, s.ConnectionState, s.SessionState)
}

func sgPersistCmd(args ...string) (string, error) {
	log.DebugLogMsg("run sg_persist with args: %s", strings.Join(args, " "))
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, "sg_persist", args...)
	result, err := c.CombinedOutput()
	return string(result), err
}

func PreemptLUN(devPath string) error {
	hostISCSIKey, err := generateISCSIKey()
	if err != nil {
		return fmt.Errorf("generate Key err:%v", err)
	}

	// 读取以注册的 iSCSI Initiator Key
	keyInfo, err := sgPersistCmd("--read-keys", devPath)
	if err != nil {
		return fmt.Errorf("read iscsi device PR keys err:%v", err)
	}
	if !strings.Contains(string(keyInfo), hostISCSIKey) {
		// 没有 Key 需要先注册
		_, err := sgPersistCmd("--out", "--register", "--param-sark="+hostISCSIKey, devPath, "--param-aptpl")
		if err != nil {
			return fmt.Errorf("register PR key err:%v", err)
		}
	}

	// 读取 iSCSI 预留信息
	reservationInfo, err := sgPersistCmd("--read-reservation", devPath)
	if err != nil {
		return fmt.Errorf("read reservation info err:%v", err)
	} else if strings.Contains(reservationInfo, "NO reservation") {
		// 未设置预留，直接获取锁
		_, err = sgPersistCmd("--out", "--reserve", "--param-rk="+hostISCSIKey, "--prout-type=3", devPath)
		if err != nil {
			return fmt.Errorf("execute reserve command err:%v", err)
		}
	} else if !strings.Contains(reservationInfo, hostISCSIKey) {
		// 已设置预留但由其他节点挂载，执行抢占
		re := regexp.MustCompile(`Key=(\w+)`)
		match := re.FindStringSubmatch(reservationInfo)
		if len(match) != 2 {
			return fmt.Errorf("get key from reservationInfo err, reservationInfo: %s", reservationInfo)
		}
		ownerKey := match[1]
		_, err = sgPersistCmd("--out",
			"--preempt",
			"--param-rk="+hostISCSIKey,
			"--param-sark="+ownerKey,
			"--prout-type=3",
			devPath,
		)
		if err != nil {
			return fmt.Errorf("execute preempt command err:%v", err)
		}
	}
	return nil
}

func generateISCSIKey() (string, error) {
	// 以 Hostname 的 md5 值前 15 位生成当前节点的 iSCSI Key
	hostname, err := os.ReadFile("/etc/hostname")
	if err != nil {
		return "", err
	}
	hasher := md5.New()
	hasher.Write(hostname)
	md5Hash := hex.EncodeToString(hasher.Sum(nil))
	iscsiKey, err := strconv.ParseInt(md5Hash[:15], 16, 64)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("0x%x", iscsiKey), nil
}
