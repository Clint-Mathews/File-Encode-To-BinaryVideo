package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"
)

const (
	dummySplitByte = "00000000"        // Used to seperate file data and file type
	blankByte      = "11111111"        // On video is decode the end is padded with blankbytes Black pixels
	outputVideo    = "binaryVideo.mp4" // Default output video name and format
	outputFile     = "decodedFile"     // Default output file name
)

var (
	fileDataBitsAsString strings.Builder
	inputfile            = flag.String("inputfile", "", "Path to file to be encoded to binary!")
	height               = flag.Int("h", 480, "Frame height of the video!")
	width                = flag.Int("w", 640, "Frame width of the video!")
	cpuprofile           = flag.String("cpuprofile", "", "write cpu profile to `file`")
)

func main() {
	flag.Parse()

	if *inputfile == "" {
		panic("Input file invalid!")
	}

	fmt.Println("File encode to BinaryVideo -> Decode back to file!")

	// Setting up cpu profiling
	if *cpuprofile != "" {
		f, err := os.Create("./profiles/" + *cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	startTime := time.Now()

	// Read file and convert to video
	readAndEncodeASCIIFileAsBinary()
	createVideoFromBitString()

	// Decode video and convert back to file
	outputByteArray := decodeVideoToBinaryString()
	file, fileType := decodeFileFromBinaryToASCII(outputByteArray)
	outputFileName := generateFileUsingDecodedBytes(file, fileType)
	fmt.Printf("Output decoded file from video: %s, Time take for encode-decode: %s\n", outputFileName, time.Since(startTime))
}

func readAndEncodeASCIIFileAsBinary() {
	file, err := os.Open(*inputfile)

	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Read file type
	st, _ := os.Stat(*inputfile)
	fileName := *inputfile
	fileType := fileName[strings.LastIndex(fileName, ".")+1:]
	fmt.Printf("Size of the file in Mb: %d bytes, file type: %s \n", st.Size(), fileType)

	// Define the size of the buffer window to read
	const windowSize = 1024
	buffer := make([]byte, windowSize)

	// Read the file in chunks
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			log.Printf("Error reading file: %v", err)
			return
		}
		if n == 0 {
			// Append DummyByte and FileType to end file data
			fileDataBitsAsString.WriteString(createBinaryAppenededString([]byte{0}))
			fileDataBitsAsString.WriteString(createBinaryAppenededString([]byte(fileType)))
			break // End of file
		}
		// Append file data
		fileDataBitsAsString.WriteString(createBinaryAppenededString(buffer[:n]))
	}
}

func createBinaryAppenededString(b []byte) (d string) {
	for _, byteData := range b {
		// Format byte as a binary string with 8 bits and append to string
		d += fmt.Sprintf("%08b", byteData)
	}
	return
}

func createVideoFromBitString() {
	frameData := createVideoFrameData()

	// Write frame data to a temporary file
	tempFile, err := os.CreateTemp("", "video_*.raw")
	if err != nil {
		fmt.Println("Error creating temporary file:", err)
	}
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write(frameData)
	if err != nil {
		fmt.Println("Error writing binary data to file:", err)
	}
	tempFile.Close()

	// Convert frames to video using FFmpeg
	// Raw video data as input, encodes it using the H.264 codec, and produces an output video file with specified settings and parameters.
	// The resulting video is encoded with ultrafast speed and high-quality output.
	// `-qp 0` sets video to have higher quality so that we have a lossless video, which helps in decoding to get back the same data
	cmd := exec.Command("ffmpeg", "-y", "-f", "rawvideo", "-pix_fmt", "rgb24", "-s", fmt.Sprintf("%dx%d", *width, *height), "-r", "24", "-i", tempFile.Name(), "-c:v", "libx264", "-preset", "ultrafast", "-qp", "0", "-pix_fmt", "yuv420p", outputVideo)
	err = cmd.Run()
	if err != nil {
		fmt.Println("Error converting binary data to video:", err)
	}

	fmt.Println("Video created successfully:", outputVideo)
}

func createVideoFrameData() []byte {
	fileDataAndFileTypeBitString := fileDataBitsAsString.String()
	// Clear unwanted strings builder!
	fileDataBitsAsString.Reset()

	// Based on number of bits we would need that many pixels and based on height and width we get the number of frames required
	numFrames := int(math.Ceil(float64(len(fileDataAndFileTypeBitString)) / float64((*height)*(*width))))

	// Each frame containing pixels requires RGB channel
	frameSize := (*width) * (*height) * 3

	// Creating a video byte array for the output data in frames
	frameData := make([]byte, numFrames*frameSize)
	count := 0
	// Loop till each frame is filled with data
	// We should not exceed the fileDataAndFileTypeBitString length and the rest of the frame would be filled with black pixels
	for i := 0; i < numFrames && len(fileDataAndFileTypeBitString) > count; i++ {
		// Offset to skip to required frame index i.e, required frame!
		offset := i * frameSize
		count = fillFrame(frameData[offset:offset+frameSize], fileDataAndFileTypeBitString, count)
	}
	return frameData
}

// Fill frame pixel RGB channel with required color based on data
func fillFrame(frame []byte, stringData string, count int) int {
	// Filling each pixel!
	for y := 0; y < *height && len(stringData) > count; y++ {
		for x := 0; x < *width && len(stringData) > count; x++ {
			// frameOffset refers to the position within the frame matrix where data requires to be written!
			// y*(*width) - Calculates the offset to the start of the row where the pixel should be written
			// (y*(*width) + x) - Calculates the offset to the column of the row where the pixel should be written
			// (y*(*width) + x) * 3 - Calculates the total offset within the frame matrix where the RGB values for the pixel at position
			frameOffset := (y*(*width) + x) * 3
			bitForFrame := 0
			// If 0 we set pixel as white else as black
			// 255,255,255 - White
			// 0,0,0 - Black
			if string(stringData[count]) == "0" {
				bitForFrame = 255
			}
			frame[frameOffset] = byte(bitForFrame)
			frame[frameOffset+1] = byte(bitForFrame)
			frame[frameOffset+2] = byte(bitForFrame)
			count++
		}
	}
	// Count is used to loop through the fileDataAndFileTypeBitString
	return count
}

func decodeVideoToBinaryString() []string {
	// Setting video file
	inputFile := outputVideo

	// Decodes video using FFmpeg, and outputs the raw video frames in RGB24
	cmd := exec.Command("ffmpeg", "-i", inputFile, "-f", "image2pipe", "-vcodec", "rawvideo", "-pix_fmt", "rgb24", "-")

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Printf("error running ffmpeg command: %v \n", err)
	}

	//  Extracts the raw video data from the out buffer
	data := out.Bytes()
	var result bytes.Buffer
	var byteStringArr []string
	// Loops through the data from start and increments by frame size
	for i := 0; i < len(data); i += (*width) * (*height) * 3 {
		// Loops from the frame start till end and increments by pixel's same color channel value.
		// i.e, We can consider the value in any of the RGB channel as all of them would have same value
		for j := i; j < i+((*width)*(*height)*3); j += 3 {
			// For each color channel value (data[j]),
			// if the value is greater than 127 (approximately halfway between 0 and 255), it appends '0'
			// Else, it appends '1'
			// This effectively converts color channel value to a binary digit
			// Reason for checking > 127 is because we see loss in video which shows that even if we appended 255 we might get a value in range of [200-255] etc
			// So we can take either a max value of 3 RGB channel or take any of the channel value!
			if data[j] > 127 {
				result.WriteByte('0')
			} else {
				result.WriteByte('1')
			}
			// Saving 8 bits together to convert the bitString to ASCII later
			if result.Len() == 8 {
				byteStringArr = append(byteStringArr, result.String())
				result.Reset()
			}
		}
	}

	// Removes unwanted blank data
	// End of the video is appended by black pixels which has no data and we should clear them out
	check := blankByte
	for check == blankByte {
		byteStringArr = byteStringArr[:len(byteStringArr)-1]
		check = byteStringArr[len(byteStringArr)-1]
	}
	return byteStringArr
}

func decodeFileFromBinaryToASCII(byteStringArr []string) (file []byte, fileType []byte) {
	// Find File data and Filetype split by dummySplitByte
	lastIndex := 0
loop:
	for i := len(byteStringArr) - 1; i >= 0; i-- {
		// Index to split file data and file type based on dummySplitByte
		if byteStringArr[i] == dummySplitByte {
			lastIndex = i
			break loop
		}
	}

	fileData := byteStringArr[:lastIndex]
	fileTypeData := byteStringArr[lastIndex+1:]

	// Find file and fileType bytes
	file = convertBinaryToASCIIByteArray(fileData)
	fileType = convertBinaryToASCIIByteArray(fileTypeData)
	return
}

func convertBinaryToASCIIByteArray(s []string) (retData []byte) {
	for _, stringData := range s {
		retData = append(retData, convertBinaryToASCII(stringData))
	}
	return
}

func convertBinaryToASCII(s string) byte {
	// Return byte value of the binary string
	num, _ := strconv.ParseInt(s, 2, 10)
	return byte(num)
}

func generateFileUsingDecodedBytes(fileDataBytes, fileTypeBytes []byte) (fileName string) {
	// Generate file using filename and fileTypeBytes
	fileName = fmt.Sprintf("%s.%s", outputFile, fileTypeBytes)
	outputFile, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer outputFile.Close()

	// Append fileDataBytes to file
	_, err = outputFile.Write(fileDataBytes)
	if err != nil {
		log.Fatal(err)
	}
	return
}

// UnOptimized function to read data from file!
func readAndEncodeASCIIFileAsBinaryUnOptimized() string {
	file, err := os.Open(*inputfile)

	if err != nil {
		panic(err)
	}
	defer file.Close()

	st, _ := os.Stat(*inputfile)
	fileName := *inputfile
	fileType := fileName[strings.LastIndex(fileName, ".")+1:]
	fmt.Printf("Size of the file in Mb: %d bytes, file type: %s \n", st.Size(), fileType)

	bufr := bufio.NewReader(file)
	fileBytes, err := io.ReadAll(bufr)
	if err != nil {
		panic(err)
	}

	fileTypeBytes := []byte(fileType)
	fileBytesAppendedString := createBinaryAppenededString(fileBytes)
	fileBytesAppendedString += dummySplitByte
	fileTypeByteString := createBinaryAppenededString(fileTypeBytes)
	fileBytesAppendedString += fileTypeByteString

	return fileBytesAppendedString
}
