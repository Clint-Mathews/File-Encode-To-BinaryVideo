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
)

var (
	inputfile  = flag.String("inputfile", "", "Path to file to be encoded to binary!")
	height     = flag.Int("h", 480, "Frame height of the video!")
	width      = flag.Int("w", 640, "Frame width of the video!")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
)

func main() {
	flag.Parse()

	if *inputfile == "" {
		panic("Input file invalid!")
	}

	fmt.Println("Lets get to work!")

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

	stringOfbitData := readAndEncodeFileAsBinary()
	outputvideo := createVideoFromEncodeData(stringOfbitData)

	outputBitArray := decodeBinaryVideo(outputvideo)
	file, fileType := decodeFileFromBinaryToASCII(outputBitArray)
	decodeASCIIAndGenerateFile(file, fileType)

}

func readAndEncodeFileAsBinary() string {
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

func convertASCIIToBinary(b byte) string {
	return fmt.Sprintf("%08b", b)
}

func createBinaryList(b []byte) (d string) {
	for _, byteData := range b {
		stringData := convertASCIIToBinary(byteData)
		d += stringData
	}
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
	outputFile, err := os.Create("output." + fileType)
	if err != nil {
		log.Fatal(err)
	}
	defer outputFile.Close()

	_, err = outputFile.Write(byteArray)
	if err != nil {
		log.Fatal(err)
	}
}

func createVideoFromEncodeData(bitData string) string {
	data, _ := createBinaryVideo(bitData)
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
	outputFile := "output.mp4"
	cmd := exec.Command("ffmpeg", "-y", "-f", "rawvideo", "-pix_fmt", "rgb24", "-s", fmt.Sprintf("%dx%d", *width, *height), "-r", "24", "-i", tempFile.Name(), "-c:v", "libx264", "-preset", "ultrafast", "-qp", "0", "-pix_fmt", "yuv420p", outputFile)
	err = cmd.Run()
	if err != nil {
		fmt.Println("Error converting binary data to video:", err)
		return ""
	}

	fmt.Println("Video created successfully:", outputFile)
	return outputFile
}

func createBinaryVideo(stringData string) ([]byte, error) {
	numFrames := int(math.Ceil(float64(len(stringData)) / float64((*height)*(*width))))
	frameSize := (*width) * (*height) * 3

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
	for i := 0; i < len(data); i += (*width) * (*height) * 3 {
		for j := i; j < i+((*width)*(*height)*3); j += 3 {
			if data[j] > 127 {
				result.WriteByte('0')
			} else {
				result.WriteByte('1')
			}
		}
	}
	return createStringArrayFromString(result.String())
}

func createStringArrayFromString(data string) (ret []string) {
	for i := 0; i < len(data) && i+8 < len(data); i += 8 {
		ret = append(ret, data[i:i+8])
	}

	check := blankByte
	for check == blankByte {
		ret = ret[:len(ret)-1]
		check = ret[len(ret)-1]
	}
	return
}
