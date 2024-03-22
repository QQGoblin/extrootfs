# -*- coding: utf-8 -*-
import argparse
import copy
import hashlib
import json
import logging
import os
import platform
import shutil
import ssl
import subprocess
import warnings

logging.basicConfig(level=logging.INFO, format='%(asctime)s %(message)s')
ssl._create_default_https_context = ssl._create_unverified_context
warnings.filterwarnings("ignore", category=DeprecationWarning)

global OUTPUT_DIR
global RAW_IMAGE_DIR
global METADATA_DIR
global MNT_DIR
global UNPACK_DIR
global ROOTFS_DIR
cpu_architecture = platform.machine()
DEFAULT_SKOPEO_COPY_SRC = "docker-daemon:"
DEFAULT_METADATA_TAR = "metadata.tar"
DEFAULT_ROOTFS_TAR = "rootfs.tar"
DEFAULT_META_LAYER_URL = "http://172.28.117.33:30080/public/others/alpine/c1d6d1b2d5a367259e6e51a7f4d1ccd66a28cc9940d6599d8a8ea9544dd4b4a8"
DEFAULT_META_LAYER_NAME = "c1d6d1b2d5a367259e6e51a7f4d1ccd66a28cc9940d6599d8a8ea9544dd4b4a8"
DEFAULT_META_ROOTFS = {
    "type": "layers",
    "diff_ids": [
        "sha256:18eb8b5891f2056b0a6c9978359916a519e8fdeec08c13c6383b922cd15fcfb2"
    ]
}
DEFAULT_META_LAYERS = [
    {
        "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
        "size": 2807669,
        "digest": "sha256:c1d6d1b2d5a367259e6e51a7f4d1ccd66a28cc9940d6599d8a8ea9544dd4b4a8"
    }
]


def create_output_dir(base):
    global OUTPUT_DIR
    global RAW_IMAGE_DIR
    global METADATA_DIR
    global MNT_DIR
    global UNPACK_DIR
    global ROOTFS_DIR

    OUTPUT_DIR = os.path.join(base, "build")
    RAW_IMAGE_DIR = os.path.join(OUTPUT_DIR, "raw")
    METADATA_DIR = os.path.join(OUTPUT_DIR, "metadata")
    MNT_DIR = os.path.join(OUTPUT_DIR, "mnt")
    UNPACK_DIR = os.path.join(OUTPUT_DIR, "unpack")
    ROOTFS_DIR = os.path.join(OUTPUT_DIR, "rootfs")

    logging.info("清理输出目录: %s", OUTPUT_DIR)
    if os.path.exists(OUTPUT_DIR):
        shutil.rmtree(OUTPUT_DIR)
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    os.makedirs(UNPACK_DIR, exist_ok=True)
    # os.makedirs(ROOTFS_DIR, exist_ok=True)
    os.makedirs(MNT_DIR, exist_ok=True)


def cal_sha256(files):
    if len(files) == 0:
        return ""
    sha256 = hashlib.sha256()
    for file in files:
        with open(file, 'rb') as f:
            while True:
                chunk = f.read(4096)
                if not chunk:
                    break
                sha256.update(chunk)
    return sha256.hexdigest()


def command(cmd, exit_on_error=True):
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    if result.returncode != 0:
        if exit_on_error:
            logging.fatal("执行命令 %s 失败, stdout: <%s>, stderr: %s", cmd, result.stdout, result.stderr)
            exit(-1)
        else:
            return
    return result.stdout


def copy_image(src, dest, opts=[]):
    logging.info("Copy Image: %s -> %s", src, dest)
    command(f"skopeo copy {' '.join(opts)} {src} {dest}")


def prepare_origin(ref):
    copy_image(DEFAULT_SKOPEO_COPY_SRC + ref, "dir:" + RAW_IMAGE_DIR)

    # 读取镜像元数据
    with open(os.path.join(RAW_IMAGE_DIR, "manifest.json"), 'r') as file:
        manifest = json.load(file)

    image_config_id = manifest["config"]["digest"].removeprefix("sha256:")

    with open(os.path.join(RAW_IMAGE_DIR, image_config_id), 'r') as file:
        image_config = json.load(file)

    return manifest, image_config


def create_meta_img(manifest, image_config):
    os.makedirs(METADATA_DIR)
    # 生成新的镜像元数据文件
    image_config["rootfs"] = DEFAULT_META_ROOTFS
    temp_image_config_name = os.path.join(METADATA_DIR, "temp.json")
    with open(os.path.join(METADATA_DIR, "temp.json"), "w") as file:
        json.dump(image_config, file)
    image_config_sha256 = cal_sha256(files=[temp_image_config_name])
    shutil.move(temp_image_config_name, os.path.join(METADATA_DIR, image_config_sha256))

    # 生成新的 manifest.json
    manifest["layers"] = DEFAULT_META_LAYERS
    manifest["config"]["digest"] = f"sha256:{image_config_sha256}"
    manifest["config"]["size"] = os.path.getsize(os.path.join(METADATA_DIR, image_config_sha256))
    with open(os.path.join(METADATA_DIR, "manifest.json"), "w") as file:
        json.dump(manifest, file)

    with open(os.path.join(METADATA_DIR, "version"), "w") as file:
        file.write("Directory Transport Version: 1.1")

    # 下载 meta 默认 layer
    command(f"curl -o {os.path.join(METADATA_DIR, DEFAULT_META_LAYER_NAME)} {DEFAULT_META_LAYER_URL}")


def create_rootfs_data(manifest):
    # 获取所有 layer 列表

    temp = os.path.join(UNPACK_DIR, "temp")
    os.makedirs(temp)

    lowerdir = [temp]
    layers = []
    for layer in manifest["layers"]:
        digest = layer["digest"].removeprefix("sha256:")
        layer_unpack_dir = os.path.join(UNPACK_DIR, digest)
        os.makedirs(layer_unpack_dir)
        layer_path = os.path.join(RAW_IMAGE_DIR, digest)
        command(f"tar -xf {layer_path} -C {layer_unpack_dir}")
        lowerdir.append(layer_unpack_dir)
        layers.append(layer_path)

    lowerdir.reverse()
    lowerdir_s = ':'.join(lowerdir)
    mount_cmd = f"mount -t overlay overlay -o lowerdir={lowerdir_s},index=off {MNT_DIR}"
    logging.info("Mount CMD: %s", mount_cmd)

    try:
        command(mount_cmd)
        # shutil.copytree(MNT_DIR, ROOTFS_DIR)
        command(f"cp -rp {MNT_DIR} {ROOTFS_DIR}")
        command(f"tar -cf {os.path.join(OUTPUT_DIR, DEFAULT_ROOTFS_TAR)} -C {OUTPUT_DIR} rootfs")
        # 由于 cp 过程中每次软连接都会新建，这会导致计算 rootfs.tar 的 sha256sum 每次都不同，因此采用其他方式计算
        rootfs_sha256 = cal_sha256(layers)

    finally:
        command(f"umount {MNT_DIR}")

    return rootfs_sha256


def create_config(mref, rootfs_sha256):
    metadata_sha256 = cal_sha256(files=[os.path.join(OUTPUT_DIR, DEFAULT_METADATA_TAR)])
    id_obj = hashlib.sha256()
    id_obj.update(f"%{metadata_sha256} {rootfs_sha256}".encode())
    ext_id = id_obj.hexdigest()
    config = {
        "id": ext_id,
        "architecture": cpu_architecture,
        "os": "linux",
        "sha256": rootfs_sha256,
        "metadata": {
            "sha256": metadata_sha256,
            "ref": mref
        }
    }

    with open(os.path.join(OUTPUT_DIR, "config.json"), "w") as file:
        json.dump(config, file)

    return ext_id


def meta_image_ref(ref):
    return f"{ref}-meta"


def create_external_img(ref, mref):
    if mref == "":
        mref = meta_image_ref(ref)

    # 导出镜像到本地目录
    manifest, image_config = prepare_origin(ref)

    # 生成 meta 镜像
    create_meta_img(copy.deepcopy(manifest), copy.deepcopy(image_config))
    copy_image(f"dir:{METADATA_DIR}",
               f"docker-archive:{os.path.join(OUTPUT_DIR, DEFAULT_METADATA_TAR)}",
               opts=["--additional-tag", mref])

    # 生成 rootfs data
    rootfs_sha256 = create_rootfs_data(manifest)

    # 输出 CONFIG 文件
    ext_id = create_config(mref, rootfs_sha256)
    with open(os.path.join(OUTPUT_DIR, "sha256"), "w") as file:
        file.write(ext_id)

    # 清理环境
    for clean_path in [METADATA_DIR, UNPACK_DIR, MNT_DIR, RAW_IMAGE_DIR, ROOTFS_DIR]:
        shutil.rmtree(clean_path)


if __name__ == "__main__":
    p = argparse.ArgumentParser(description="""External Images 构建工具:

     示例： python3 create-extimg.py -ref registry.lqingcloud.cn/cilium/cilium:12.13.1
     
     执行环境需要满足以下条件：
     - 安装 docker 并且已经拉取目标镜像到本地
     - 安装 skopeo 工具
     """,
                                formatter_class=argparse.RawTextHelpFormatter)
    p.add_argument("-ref",
                   help="需要转化的原始镜像名称",
                   dest="ref",
                   type=str,
                   default="")
    p.add_argument("-mref",
                   help="生成 External Metadata Image 的名称",
                   dest="mref",
                   type=str,
                   default="")
    p.add_argument("-src",
                   help="原始镜像源",
                   dest="src",
                   type=str,
                   default="")

    args = p.parse_args()
    if args.ref == "":
        logging.error("请指定要转换的镜像")
        exit(-1)
    if args.src != "":
        DEFAULT_SKOPEO_COPY_SRC = args.src

    create_output_dir(os.getcwd())
    create_external_img(args.ref, args.mref)
