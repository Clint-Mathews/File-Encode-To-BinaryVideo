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
)

const (
	dummySplitByte = "00000000"
	blankByte      = "11111111"
	outputVideo    = "output.mp4"
	outputFile     = "output"
)

var (
	fileDataInBits strings.Builder
	inputfile      = flag.String("inputfile", "", "Path to file to be encoded to binary!")
	height         = flag.Int("h", 480, "Frame height of the video!")
	width          = flag.Int("w", 640, "Frame width of the video!")
	cpuprofile     = flag.String("cpuprofile", "", "write cpu profile to `file`")
)

func main() {
	flag.Parse()

	if *inputfile == "" {
		panic("Input file invalid!")
	}

	fmt.Println("File encode to BinaryVideo -> Decode back to file!")

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

	// Read file and convert to video
	readAndEncodeFileAsBinary()
	outputvideo := createVideoFromEncodeData()

	// Decode video and convert back to file
	outputBitArray := decodeBinaryVideo(outputvideo)
	file, fileType := decodeFileFromBinaryToASCII(outputBitArray)
	decodeASCIIAndGenerateFile(file, fileType)
}

func readAndEncodeFileAsBinary() {
	file, err := os.Open(*inputfile)

	if err != nil {
		panic(err)
	}
	defer file.Close()

	st, _ := os.Stat(*inputfile)
	fileName := *inputfile
	fileType := fileName[strings.LastIndex(fileName, ".")+1:]
	fmt.Printf("Size of the file in Mb: %d bytes, file type: %s \n", st.Size(), fileType)

	// Define the size of the window
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
			fileDataInBits.WriteString(createBinaryList([]byte{0}))
			fileDataInBits.WriteString(createBinaryList([]byte(fileType)))
			break // End of file
		}
		// Append file data
		fileDataInBits.WriteString(createBinaryList(buffer[:n]))
	}
}

func createBinaryList(b []byte) (d string) {
	for _, byteData := range b {
		d += fmt.Sprintf("%08b", byteData)
	}
	return
}

func decodeFileFromBinaryToASCII(data []string) (file []byte, fileType []byte) {
	lastIndex := 0
loop:
	for i := len(data) - 1; i >= 0; i-- {
		if data[i] == dummySplitByte {
			lastIndex = i
			break loop
		}
	}

	fileData := data[:lastIndex]
	fileTypeData := data[lastIndex+1:]

	file = createASCIIList(fileData)
	fileType = createASCIIList(fileTypeData)
	return
}

func convertBinaryToASCII(s string) byte {
	num, _ := strconv.ParseInt(s, 2, 10)
	return byte(num)
}

func createASCIIList(s []string) (retData []byte) {
	for _, stringData := range s {
		retData = append(retData, convertBinaryToASCII(stringData))
	}
	return
}

func decodeASCIIAndGenerateFile(byteArray, fileTypeBytes []byte) {
	fileType := string(fileTypeBytes)
	outputFile, err := os.Create(fmt.Sprintf("%s.%s", outputFile, fileType))
	if err != nil {
		log.Fatal(err)
	}
	defer outputFile.Close()

	_, err = outputFile.Write(byteArray)
	if err != nil {
		log.Fatal(err)
	}
}

func createVideoFromEncodeData() string {
	data, _ := createBinaryVideo()

	// Write binary data to a temporary file
	tempFile, err := os.CreateTemp("", "video_*.raw")
	if err != nil {
		fmt.Println("Error creating temporary file:", err)
		return ""
	}
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write(data)
	if err != nil {
		fmt.Println("Error writing binary data to file:", err)
		return ""
	}
	tempFile.Close()

	// Convert binary data to video using FFmpeg

	cmd := exec.Command("ffmpeg", "-y", "-f", "rawvideo", "-pix_fmt", "rgb24", "-s", fmt.Sprintf("%dx%d", *width, *height), "-r", "24", "-i", tempFile.Name(), "-c:v", "libx264", "-preset", "ultrafast", "-qp", "0", "-pix_fmt", "yuv420p", outputVideo)
	err = cmd.Run()
	if err != nil {
		fmt.Println("Error converting binary data to video:", err)
		return ""
	}

	fmt.Println("Video created successfully:", outputVideo)
	return outputVideo
}

func createBinaryVideo() ([]byte, error) {
	stringData := fileDataInBits.String()
	// Clear unwanted strings builder!
	fileDataInBits.Reset()
	// Based on number of bits we would need that many pixels and based on height and width we get the number of frames required
	numFrames := int(math.Ceil(float64(len(stringData)) / float64((*height)*(*width))))
	// Each frame which has pixels requires RGB channel
	frameSize := (*width) * (*height) * 3

	// Creating a video byte array for the output data in frames
	data := make([]byte, numFrames*frameSize)
	count := 0
	for i := 0; i < numFrames && len(stringData) > count; i++ {
		offset := i * frameSize
		count = fillFrame(data[offset:offset+frameSize], stringData, count)
	}

	return data, nil
}

func fillFrame(frame []byte, stringData string, count int) int {
	for y := 0; y < *height && len(stringData) > count; y++ {
		for x := 0; x < *width && len(stringData) > count; x++ {
			frameOffset := (y*(*width) + x) * 3
			bitForFrame := 0
			if string(stringData[count]) == "0" {
				bitForFrame = 255
			}
			frame[frameOffset] = byte(bitForFrame)
			frame[frameOffset+1] = byte(bitForFrame)
			frame[frameOffset+2] = byte(bitForFrame)
			count++
		}
	}
	return count
}

func decodeBinaryVideo(inputFile string) []string {
	cmd := exec.Command("ffmpeg", "-i", inputFile, "-f", "image2pipe", "-vcodec", "rawvideo", "-pix_fmt", "rgb24", "-")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Printf("error running ffmpeg command: %v \n", err)
	}

	data := out.Bytes()
	var result bytes.Buffer
	var byteStrings []string
	for i := 0; i < len(data); i += (*width) * (*height) * 3 {
		for j := i; j < i+((*width)*(*height)*3); j += 3 {
			if data[j] > 127 {
				result.WriteByte('0')
			} else {
				result.WriteByte('1')
			}
			if result.Len() == 8 {
				byteStrings = append(byteStrings, result.String())
				result.Reset()
			}
		}
	}

	// Remove unwanted blank data
	check := blankByte
	for check == blankByte {
		byteStrings = byteStrings[:len(byteStrings)-1]
		check = byteStrings[len(byteStrings)-1]
	}
	return byteStrings
}

// UnOptimized function to read data from file!
func readAndEncodeFileAsBinaryUnOptimized() string {
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
	fileBytesAppendedString := createBinaryList(fileBytes)
	fileBytesAppendedString += dummySplitByte
	fileTypeByteString := createBinaryList(fileTypeBytes)
	fileBytesAppendedString += fileTypeByteString

	return fileBytesAppendedString
}
