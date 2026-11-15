from PIL import Image, ImageDraw, ImageFont
from pathlib import Path
import struct
root=Path(__file__).parent
font='/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf'
for enabled in [True,False]:
    size=32; scale=4; s=size*scale
    img=Image.new('RGBA',(s,s),(0,0,0,0))
    draw=ImageDraw.Draw(img)
    bg=(3,145,125,255) if enabled else (118,124,132,255)
    draw.rounded_rectangle((7,7,s-8,s-8),radius=23,fill=bg)
    draw.rounded_rectangle((12,12,s-13,s-13),radius=21,outline=(255,255,255,70),width=3)
    f=ImageFont.truetype(font,78)
    box=draw.textbbox((0,0),'D',font=f)
    w,h=box[2]-box[0],box[3]-box[1]
    draw.text(((s-w)/2-box[0]-1,(s-h)/2-box[1]-5),'D',fill=(255,255,255,255),font=f,stroke_width=0)
    if enabled:
        draw.ellipse((s-36,s-37,s-7,s-8),fill=(74,217,129,255),outline=(255,255,255,255),width=4)
    else:
        draw.ellipse((s-36,s-37,s-7,s-8),fill=(174,175,177,255),outline=(243,243,243,255),width=4)
    img=img.resize((size,size),Image.Resampling.LANCZOS)
    img.save(root / ('tray_on_preview.png' if enabled else 'tray_off_preview.png'))
    # 32bpp uncompressed icon DIB (not whole ICO): BITMAPINFOHEADER, XOR, AND.
    # biHeight=2*height includes the monochrome AND transparency mask.
    h=struct.pack('<IiiHHIIiiII',40,size,size*2,1,32,0,size*size*4,0,0,0,0)
    xor=bytearray()
    for y in range(size-1,-1,-1):
        for x in range(size):
            r,g,b,a=img.getpixel((x,y)); xor.extend((b,g,r,a))
    mask=bytearray()
    for y in range(size-1,-1,-1):
        line=bytearray([0]*4)
        for x in range(size):
            if img.getpixel((x,y))[3]==0:
                line[x//8] |= 1 << (7-(x%8))
        mask.extend(line)
    blob=h+xor+mask
    (root / ('tray_on.bin' if enabled else 'tray_off.bin')).write_bytes(blob)
    print('generated',('enabled' if enabled else 'disabled'),'icon',len(blob),'bytes')
