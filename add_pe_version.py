"""Add Windows VERSIONINFO resource to a PE32+ without changing executable code.
This script is used on the compiled file and doesn't require Windows SDK tools.
"""
from pathlib import Path
import struct
import sys


def align(x, n):
    return (x + n - 1) // n * n


def block(key, value=b'', children=(), typ=1, value_len=None):
    # VS_VERSIONINFO block: WORD length; WORD valueLength; WORD type; WCHAR key[]; DWORD align
    k = key.encode('utf-16le') + b'\0\0'
    head = struct.pack('<HHH', 0, len(value)//2 if typ == 1 else len(value), typ) + k
    data = bytearray(head)
    data.extend(b'\0' * (-len(data) % 4))
    data += value
    for child in children:
        data.extend(b'\0' * (-len(data) % 4))
        data.extend(child)
    struct.pack_into('<H', data, 0, len(data))
    struct.pack_into('<H', data, 2, len(value)//2 if value_len is None and typ == 1 else (value_len if value_len is not None else len(value)))
    return bytes(data)


def text_entry(key, val):
    return block(key, (val+'\0').encode('utf-16le'), typ=1)


def version_data():
    major, minor, patch, build = 1, 0, 0, 0
    ffi = struct.pack('<13I', 0xFEEF04BD, 0x00010000,
                      major<<16 | minor, patch<<16 | build,
                      major<<16 | minor, patch<<16 | build,
                      0x3F, 0x00, 0x40004, 0x1, 0, 0, 0)
    entries = [
        text_entry('CompanyName', 'Lee Sang Il'),
        text_entry('FileDescription', 'English word tooltips and Gemini Korean-English sentence translation | Lee Sang Il | Free use'),
        text_entry('FileVersion', '1.0.0.0'),
        text_entry('InternalName', '231Dictionary'),
        text_entry('OriginalFilename', '231Dictionary.exe'),
        text_entry('ProductName', '231Dictionary'),
        text_entry('ProductVersion', '1.0.0.0'),
        text_entry('LegalCopyright', 'Copyright © 2026 Lee Sang Il. Free personal and business use.'),
        text_entry('Comments', 'Created by Lee Sang Il. Free for personal and business use. Not official TDM software.'),
    ]
    strings = block('040904B0', children=entries, typ=1)
    sfi = block('StringFileInfo', children=[strings], typ=1)
    translation = block('Translation', struct.pack('<HH', 0x0409, 0x04B0), typ=0)
    vfi = block('VarFileInfo', children=[translation], typ=1)
    return block('VS_VERSION_INFO', ffi, children=[sfi, vfi], typ=0, value_len=len(ffi))


def patch_pe(path):
    data = bytearray(Path(path).read_bytes())
    e_lfanew = struct.unpack_from('<I', data, 0x3C)[0]
    assert data[e_lfanew:e_lfanew+4] == b'PE\0\0'
    co = e_lfanew+4
    assert struct.unpack_from('<H',data,co)[0] == 0x8664
    num = struct.unpack_from('<H',data,co+2)[0]
    optsize = struct.unpack_from('<H',data,co+16)[0]
    opt=co+20
    assert struct.unpack_from('<H',data,opt)[0] == 0x20B
    sections = opt+optsize
    header_size=struct.unpack_from('<I',data,opt+60)[0]
    assert sections+(num+1)*40 < header_size, 'No space for new section header'
    file_align = struct.unpack_from('<I',data,opt+36)[0]
    section_align = struct.unpack_from('<I',data,opt+32)[0]
    image_size = struct.unpack_from('<I',data,opt+56)[0]
    rva = align(image_size, section_align)
    existing_res_rva = struct.unpack_from('<I', data, opt+112+2*8)[0]
    assert existing_res_rva == 0, 'Executable already has resources, refusing to overwrite'
    vs = version_data()
    buf = bytearray(0x58)
    # IMAGE_RESOURCE_DIRECTORY root, type 16, name 1, lang 0x0412
    struct.pack_into('<IIHHHH', buf, 0x00, 0, 0, 0, 0, 0, 1)
    struct.pack_into('<II',buf,0x10,16,0x80000018)
    struct.pack_into('<IIHHHH',buf,0x18,0,0,0,0,0,1)
    struct.pack_into('<II',buf,0x28,1,0x80000030)
    struct.pack_into('<IIHHHH',buf,0x30,0,0,0,0,0,1)
    struct.pack_into('<II',buf,0x40,0x0409,0x48)
    struct.pack_into('<IIII',buf,0x48,rva+0x58,len(vs),1200,0)
    buf.extend(vs)
    raw_start = align(len(data),file_align)
    if raw_start>len(data): data.extend(b'\0'*(raw_start-len(data)))
    rawsize=align(len(buf),file_align)
    # IMAGE_SECTION_HEADER
    offset=sections+num*40
    section = struct.pack('<8sIIIIIIHHI', b'.rsrc\0\0\0', len(buf), rva,
                          rawsize, raw_start, 0, 0, 0, 0, 0x40000040)
    data[offset:offset+40]=section
    struct.pack_into('<H',data,co+2,num+1)
    struct.pack_into('<I',data,opt+56,align(rva+len(buf),section_align))
    struct.pack_into('<II',data,opt+112+2*8,rva,len(buf))
    data.extend(buf)
    data.extend(b'\0'*(rawsize-len(buf)))
    Path(path).write_bytes(data)
    print('Appended .rsrc',hex(rva),'version bytes',len(vs),'file',len(data))


if __name__=='__main__':
    patch_pe(sys.argv[1])
