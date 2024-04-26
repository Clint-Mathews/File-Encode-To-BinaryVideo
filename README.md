
# File-To-BinaryVideo-BackTo-File

Encode any file to binary video, which can then be decoded back to the original file!

## Prerequisites 

- Golang, version: `1.22.1`
```bash
brew install go
```
- ffmpeg, version: `6.1.1`
```bash
brew install ffmpeg
```

## Run Locally

- Clone the project
- Go to the project directory
```bash
cd File-To-BinaryVideo-BackTo-File
```
- Run the program

```bash
  go run . -inputfile "inputfilename.filetype" -cpuprofile cpuprof.prof
```

### Program input parameters:
Example:
```bash
  go run . -inputfile test.png -cpuprofile cpuprof.prof -h 1080 -w 1920
```

#### Required parameters:
- inputfile: Path to file to be encoded to binary. Eg: `-inputfile test.png`
- cpuprofile: File name to save cpu profiling. Eg: `-cpuprofile cpuprof.prof`
#### Optional parameters:
- h: Frame height of the video. Eg: `-h 1080`
- w: Frame width of the video. Eg: `-h 1920`

### Program output:

- `binaryVideo.mp4`: Binary video of the file.
- `decodedFile.*filetype`: Decoded file from the binary video.
- The programme also prints out the encode-decode execution time.


## Demo

https://github.com/Clint-Mathews/File-To-BinaryVideo-BackTo-File/assets/19289251/5eee006a-8912-43ec-8d61-a8ef5451ed94

## Lessons Learned

- How file/data is stored in memeory and how to manipulate that!
- Reading file as chunks rather than reading the entire file to memory
- Better use of `strings.Builder`, `bytes.Buffer`
- File I/O operations
- ASCII <-> binary converstions 
- Video creation using ffmpeg
- RGB Channels, frames and how to create images using the same

## Links

[![linkedin](https://img.shields.io/badge/linkedin-0A66C2?style=for-the-badge&logo=linkedin&logoColor=white)](https://www.linkedin.com/in/clint-mathews/)

## Features

- Can encode/decode any file type but it does not re-decode sound, eg: image, zip, pdf, etc.
- Since we are creating lossless image the video does take a bit of space, so this is great for files less than 50Mb. For reference: 10Mb file took ~5-6sec for whole encode-decode process.


## Optimizations

- Using file streaming instead of reading everything into memory 
- Using flag to receive input parameters from CLI
- Refactored functions


## Limitations
- Since the implemetation is depended on video frames to extract data, lossless videos are required. Decoding videos with loss would result in malformed data.
- Implemetation does have streaming of files but during the entire process there are a lot of file read-write operations happening and video files would have a considerable size based on resolution and size of file.
- Time and space for processing are directly linked to file size, greater the size longer the binary video.
- Video files would loss audio in the process but the video is retained correctly. 
