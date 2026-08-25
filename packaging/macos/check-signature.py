#!/usr/bin/env python3
"""Mach-O 실행파일에 ad-hoc 코드 서명이 붙었는지 확인한다.

왜 이걸 잰다
애플 실리콘(arm64) 커널은 서명 없는 실행파일을 로드하지 않는다 — 실행하면
"killed: 9" 한 줄만 나오고 이유가 어디에도 안 남는다. Go 내부 링커가 크로스
빌드에서도 ad-hoc 서명을 붙여주지만 그건 툴체인의 성질이지 우리 계약이 아니다.
빌드에서 확인하면 그것이 깨진 날 맥이 아니라 여기서 알게 된다.
"""
import struct
import sys

LC_CODE_SIGNATURE = 0x1D
CSMAGIC_EMBEDDED_SIGNATURE = 0xFADE0CC0


def check(path):
    data = open(path, "rb").read()
    magic, cputype = struct.unpack_from("<II", data, 0)
    if magic != 0xFEEDFACF:
        raise SystemExit(f"{path}: Mach-O 64 비트가 아니다 (magic={magic:#x})")
    ncmds = struct.unpack_from("<I", data, 16)[0]

    off = 32
    for _ in range(ncmds):
        cmd, cmdsize = struct.unpack_from("<II", data, off)
        if cmd == LC_CODE_SIGNATURE:
            dataoff, datasize = struct.unpack_from("<II", data, off + 8)
            sup = struct.unpack_from(">I", data, dataoff)[0]
            if sup != CSMAGIC_EMBEDDED_SIGNATURE:
                raise SystemExit(
                    f"{path}: 서명 블록의 매직이 이상하다 ({sup:#x})")
            print(f"   ok  {path}  서명 {datasize} 바이트 (ad-hoc)")
            return
        off += cmdsize

    raise SystemExit(
        f"{path}: LC_CODE_SIGNATURE 가 없다\n"
        "  애플 실리콘에서 이 파일은 killed: 9 로 죽는다.\n"
        "  맥에서 `codesign -s - <파일>` 로 ad-hoc 서명을 붙이거나,\n"
        "  ad-hoc 서명을 붙이는 Go 툴체인으로 빌드하라.")


if __name__ == "__main__":
    for p in sys.argv[1:]:
        check(p)
