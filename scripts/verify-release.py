#!/usr/bin/env python3
"""Verify Sharp's four release archives and smoke-test the native package."""
import hashlib
import pathlib
import platform
import subprocess
import sys
import tarfile
import tempfile


def main():
    dist = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else "dist")
    archives = sorted(dist.glob("sharp_*.tar.gz"))
    targets = {f"{os}_{arch}" for os in ("darwin", "linux") for arch in ("amd64", "arm64")}
    actual = {"_".join(path.name.removesuffix(".tar.gz").rsplit("_", 2)[-2:]) for path in archives}
    if len(archives) != 4 or actual != targets:
        raise ValueError(f"expected one archive per {sorted(targets)}, got {[p.name for p in archives]}")
    checksums = {}
    for line in (dist / "checksums.txt").read_text().splitlines():
        digest, name = line.split(maxsplit=1)
        name = name.lstrip("*")
        if name in checksums or pathlib.Path(name).name != name:
            raise ValueError(f"invalid checksum entry: {name}")
        checksums[name] = digest
        data = (dist / name).read_bytes()
        if hashlib.sha256(data).hexdigest() != digest:
            raise ValueError(f"checksum mismatch: {name}")
    machine = {"x86_64": "amd64", "aarch64": "arm64"}.get(platform.machine().lower(), platform.machine().lower())
    native = f"{platform.system().lower()}_{machine}"
    tested_native = False
    for archive in archives:
        if archive.name not in checksums:
            raise ValueError(f"archive missing checksum: {archive.name}")
        with tarfile.open(archive, "r:gz") as package:
            members = {member.name: member for member in package.getmembers()}
            required = {"sharp", "LICENSE", "README.md", "docs/README.zh-CN.md"}
            if not required.issubset(members):
                raise ValueError(f"missing package files in {archive.name}: {required - members.keys()}")
            binary = members["sharp"]
            if not binary.isfile() or not binary.mode & 0o111:
                raise ValueError(f"sharp is not executable in {archive.name}")
            if archive.name.endswith(f"_{native}.tar.gz"):
                with tempfile.TemporaryDirectory(prefix="sharp-release-") as temp:
                    executable = pathlib.Path(temp) / "sharp"
                    executable.write_bytes(package.extractfile(binary).read())
                    executable.chmod(0o700)
                    version = subprocess.check_output([str(executable), "version"], text=True)
                    expected_version = archive.name.removeprefix("sharp_").removesuffix(f"_{native}.tar.gz")
                    if f"version: {expected_version}\n" not in version:
                        raise ValueError("packaged version does not match archive name")
                    for field in ("commit:", "built:"):
                        if field not in version:
                            raise ValueError(f"packaged binary missing {field}")
                    encoded = subprocess.check_output([str(executable), "url", "encode", "--mode", "component", "^@#$ a+b"], text=True).strip()
                    if encoded != "%5E%40%23%24%20a%2Bb":
                        raise ValueError("packaged URL command smoke failed")
                    print(version.strip())
                    tested_native = True
        print(f"Verified {archive.name}")
    if not tested_native:
        raise ValueError(f"no package available for host {native}")


if __name__ == "__main__":
    main()
