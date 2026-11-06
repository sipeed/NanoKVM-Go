from PIL import Image
import numpy as np
import cv2

def generate_mjpeg_data(width, height, quality):
    # 创建一个一半是白色一半是黑色的图像
    image = np.zeros((height, width, 3), dtype=np.uint8)
    image[:, :width//2] = [255, 255, 255]  # 白色
    image[:, width//2:] = [0, 0, 0]       # 黑色

    # 将图像编码为MJPEG格式
    _, buffer = cv2.imencode('.jpg', image, [int(cv2.IMWRITE_JPEG_QUALITY), quality])
    mjpeg_data = buffer.tobytes()

    # 将图像保存为JPEG文件到当前目录
    image_pil = Image.fromarray(cv2.cvtColor(image, cv2.COLOR_BGR2RGB))
    image_pil.save('generated_image.jpg')

    return mjpeg_data

# 生成MJPEG数据
width = 1920
height = 1080
quality = 45
mjpeg_data = generate_mjpeg_data(width, height, quality)

# 将MJPEG数据转换为C语言中的字节数组
mjpeg_data_hex = ', '.join(f'0x{byte:02x}' for byte in mjpeg_data)

# 将字节数组写入C源文件中
with open('mjpeg_data.h', 'w') as f:
    f.write(f'#define MJPEG_DATA_SIZE {len(mjpeg_data)}\n')
    f.write('static const uint8_t mjpeg_data[] = {\n')
    f.write(f'    {mjpeg_data_hex}\n')
    f.write('};\n')

print(f'MJPEG数据已生成并写入mjpeg_data.h，大小为 {len(mjpeg_data)} 字节')
print(f'图像已保存为 generated_image.jpg')
