#!/bin/bash
# Chạy script này sau mỗi lần boot Jetson (vì /mnt/ram là RAM, mất khi tắt nguồn)
# Cần có mạng internet
set -e

echo "[1/6] Mount RAM disk..."
mkdir -p /mnt/ram
mount -t tmpfs -o size=3G tmpfs /mnt/ram 2>/dev/null || true
mkdir -p /mnt/ram/tmp            # AFTER the mount (mount wipes pre-created dirs)
export TMPDIR=/mnt/ram/tmp
# /tmp lives on the (failing) microSD → back it with RAM so NO temp write hits the card
mount -t tmpfs -o size=512M tmpfs /tmp 2>/dev/null || true
chmod 1777 /tmp 2>/dev/null || true

echo "[2/6] Download Python packages..."
cd /mnt/ram
wget -q "http://ports.ubuntu.com/ubuntu-ports/pool/universe/p/python3.8/libpython3.8-stdlib_3.8.0-3ubuntu1~18.04.2_arm64.deb" -O py38stdlib.deb
wget -q "http://ports.ubuntu.com/ubuntu-ports/pool/universe/p/python3.8/libpython3.8-minimal_3.8.0-3ubuntu1~18.04.2_arm64.deb" -O py38min.deb
wget -q "http://ports.ubuntu.com/ubuntu-ports/pool/universe/p/python3.8/python3.8-minimal_3.8.0-3ubuntu1~18.04.2_arm64.deb" -O py38binpkg.deb
wget -q "https://bootstrap.pypa.io/pip/3.8/get-pip.py" -O get-pip.py

echo "[3/6] Extract stdlib..."
mkdir -p py38x py38binx
dpkg-deb -x py38stdlib.deb py38x
dpkg-deb -x py38min.deb py38x
dpkg-deb -x py38binpkg.deb py38binx

export PYTHONHOME=/mnt/ram/py38x/usr

# Patch distutils (Ubuntu tách distutils khỏi stdlib deb → tải thủ công từ CPython)
BASE="https://raw.githubusercontent.com/python/cpython/v3.8.0/Lib/distutils"
DIST="/mnt/ram/py38x/usr/lib/python3.8/distutils"
rm -rf $DIST/__pycache__
for f in __init__.py archive_util.py bcppcompiler.py ccompiler.py cmd.py config.py \
          core.py cygwinccompiler.py debug.py dep_util.py dir_util.py dist.py \
          errors.py extension.py fancy_getopt.py file_util.py filelist.py log.py \
          spawn.py sysconfig.py text_file.py unixccompiler.py util.py version.py \
          _msvccompiler.py msvc9compiler.py msvccompiler.py versionpredicate.py; do
    wget -q "$BASE/$f" -O "$DIST/$f"
done
# distutils.command subpackage — BẮT BUỘC cho get-pip (distutils.command.install)
mkdir -p "$DIST/command"
for f in __init__.py bdist.py bdist_dumb.py bdist_rpm.py build.py build_clib.py \
          build_ext.py build_py.py build_scripts.py check.py clean.py config.py \
          install.py install_data.py install_egg_info.py install_headers.py \
          install_lib.py install_scripts.py register.py sdist.py upload.py; do
    wget -q "$BASE/command/$f" -O "$DIST/command/$f"
done

echo "[4/6] Install pip..."
export PIP_NO_CACHE_DIR=1
/mnt/ram/py38binx/usr/bin/python3.8 /mnt/ram/get-pip.py \
    --prefix /mnt/ram/pyenv --no-cache-dir

echo "[5/6] Install AI packages (~10 phut)..."
export PYTHONPATH=/mnt/ram/pyenv/lib/python3.8/site-packages
# opencv-python (FULL) for the kiosk window (imshow). For pure headless you can
# swap back to opencv-python-headless; full also works headless (just no window).
/mnt/ram/py38binx/usr/bin/python3.8 -m pip install \
    --prefix /mnt/ram/pyenv --no-cache-dir \
    numpy==1.23.5 opencv-python onnxruntime==1.16.3 \
    faiss-cpu==1.7.4 requests==2.31.0 \
    albumentations==1.3.1 easydict prettytable onnx tqdm Pillow \
    scikit-image scipy scikit-learn

# Install insightface (không build Cython)
wget -q "https://files.pythonhosted.org/packages/source/i/insightface/insightface-0.7.3.tar.gz" \
    -O /mnt/ram/insightface.tar.gz
tar xzf /mnt/ram/insightface.tar.gz -C /mnt/ram
cp -r /mnt/ram/insightface-0.7.3/insightface /mnt/ram/pyenv/lib/python3.8/site-packages/

# Patch insightface (bỏ Cython extension không cần thiết)
/mnt/ram/py38binx/usr/bin/python3.8 - << 'PYEOF'
import os
fixes = [
    ('insightface/app/__init__.py',
     'from .mask_renderer import *',
     'try:\n    from .mask_renderer import *\nexcept Exception:\n    pass'),
]
base = '/mnt/ram/pyenv/lib/python3.8/site-packages'
for fname, old, new in fixes:
    fp = os.path.join(base, fname)
    if os.path.exists(fp):
        with open(fp) as f: c = f.read()
        with open(fp, 'w') as f: f.write(c.replace(old, new))

for root, dirs, files in os.walk(os.path.join(base, 'insightface')):
    for fn in files:
        if not fn.endswith('.py'): continue
        fp = os.path.join(root, fn)
        with open(fp) as f: c = f.read()
        if 'mesh_core_cython' in c:
            c = c.replace(
                'from .cython import mesh_core_cython',
                'try:\n    from .cython import mesh_core_cython\nexcept Exception:\n    mesh_core_cython = None')
            with open(fp, 'w') as f: f.write(c)
print("Patched insightface OK")
PYEOF

echo "[6/6] Test import..."
NO_ALBUMENTATIONS_UPDATE=1 \
/mnt/ram/py38binx/usr/bin/python3.8 -c "
from insightface.app import FaceAnalysis
print('Setup OK — san sang chay service')
"
