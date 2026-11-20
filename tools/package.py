"""Build the Windows and Linux amd64 portable releases."""
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import zipfile

ROOT = Path(__file__).resolve().parents[1]
DIST = ROOT / "dist"
GO = os.environ.get("GO", "go")
APP_ID = "dev.goask.GoAsk"


def go(*args, env=None):
    return subprocess.check_output([GO, *args], cwd=ROOT, env=env, text=True)


def notices(destination, env):
    destination.mkdir(parents=True, exist_ok=True)
    modules = go("list", "-deps", "-f", "{{with .Module}}{{.Path}} {{.Dir}}{{end}}", ".", env=env)
    for line in sorted(set(modules.splitlines())):
        if not line or line.startswith("goask "):
            continue
        name, path = line.split(" ", 1)
        folder = Path(path)
        for f in folder.iterdir():
            if f.is_file() and f.name.upper().startswith(("LICENSE", "NOTICE", "COPYING")):
                shutil.copyfile(f, destination / (name.replace("/", "_") + "-" + f.name))
        if name == "github.com/nsgo-dev/nsgo":
            shutil.copyfile(folder / "icons/lucide/LICENSE", destination / "Lucide-LICENSE")
    shutil.copyfile(ROOT / "internal/data/dictionary/LICENSE", destination / "ECDICT-LICENSE")
    shutil.copyfile(Path(go("env", "GOROOT").strip()) / "LICENSE", destination / "Go-LICENSE")


def package(platform):
    name = "go-ask-" + platform + "-amd64"
    DIST.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(dir=DIST) as staging:
        folder = Path(staging) / name
        folder.mkdir()
        windows = platform == "windows"
        binary = folder / ("go-ask.exe" if windows else "go-ask")
        env = dict(os.environ, GOOS=platform, GOARCH="amd64", CGO_ENABLED="0")
        release = "github.com/nsgo-dev/nsgo/core"
        flags = f"-s -w -X {release}.appID={APP_ID} -X '{release}.appName=Go Ask' -X {release}.appVersion=0.1.0 -X {release}.appRelease=1"
        if windows:
            flags += " -H=windowsgui"
        go("build", "-trimpath", "-ldflags", flags, "-o", str(binary), ".", env=env)
        shutil.copy(ROOT / "README.md", folder)
        notices(folder / "licenses", env)
        if windows:
            archive = Path(staging) / (name + ".zip")
            with zipfile.ZipFile(archive, "w", zipfile.ZIP_DEFLATED) as z:
                for f in sorted(folder.rglob("*")):
                    if f.is_file():
                        z.write(f, f.relative_to(staging))
        else:
            shutil.copy(ROOT / "assets/icon.png", folder / (APP_ID + ".png"))
            shutil.copy(ROOT / "assets" / (APP_ID + ".desktop"), folder)
            archive = Path(staging) / (name + ".tar.gz")
            with tarfile.open(archive, "w:gz") as tar:
                tar.add(folder, arcname=name)
        # Preserve the unpacked developer build alongside the portable archive.
        shutil.copytree(folder, DIST / name, dirs_exist_ok=True)
        archive = archive.replace(DIST / archive.name)
    print(archive, flush=True)


if __name__ == "__main__":
    for target in ("linux", "windows"):
        package(target)
