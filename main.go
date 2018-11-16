package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/jinzhu/now"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

type readwrite struct {
	r aws.WriteAtBuffer
}

type Request struct {
	Bucket string `json:"bucketname"`
	File   string `json:"filename"`
}

func HandleRequest(ctx context.Context, req Request) {

	//TODO: https://github.com/awsdocs/aws-doc-sdk-examples/blob/master/go/example_code/s3/upload_arbitrary_sized_stream.go
	// https://docs.aws.amazon.com/sdk-for-go/api/aws/#WriteAtBuffer
	// https://docs.aws.amazon.com/sdk-for-go/api/service/s3/s3manager/#UploadInput
	// https://github.com/awsdocs/aws-doc-sdk-examples/blob/master/go/example_code/lambda/aws-go-sdk-lambda-example-create-function.go
	// https://stackoverflow.com/questions/37469912/how-to-use-less-memory-when-sending-large-files-to-amazon-s3-via-golang-sdk
	// https://docs.aws.amazon.com/AWSJavaScriptSDK/latest/AWS/S3.html#putObject-property
	// https://stackoverflow.com/questions/43595911/how-to-save-data-streams-in-s3-aws-sdk-go-example-not-working
	// https://github.com/minio/minio-go/blob/master/examples/s3/putobject.go
	// https://gist.github.com/ehernandez-xk/e151b69f5734c8e2d7e7347d79966bb9

	fname := req.File
	bname := req.Bucket
	log.Println("Finding:", fname, "  From bucket:", bname)

	//create sessions
	log.Println("Creating new session")
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)

	downloader := s3manager.NewDownloader(sess)
	log.Println("session created, attempting to create files in lambda")
	inFile, err := os.Create("/tmp/in_" + fname)
	if nil != err {
		log.Fatal(err)
		exitErrorf("Unable to open ", inFile, " %q, %v", err)
	} else {
		log.Println("created infile " + inFile.Name())
	}
	outFile, err := os.Create("/tmp/out_" + fname)
	if nil != err {
		log.Fatal(err)
		exitErrorf("Unable to open ", outFile, " %q, %v", err)
	} else {
		log.Println("created outfile " + outFile.Name())
	}

	//log.Println("files created, setting permissions")
	//if err := inFile.Chmod(777); err != nil {
	//	log.Fatal(err)
	//} else {
	//	if err := outFile.Chmod(777); err != nil {
	//		log.Fatal(err)
	//	} else {
	//		log.Println("successfully set permissions")
	//	}
	//}
	defer inFile.Close()
	defer outFile.Close()

	log.Println("Attempting to download file")

	numBytes, err := downloader.Download(inFile,
		&s3.GetObjectInput{
			Bucket: aws.String(bname),
			Key:    aws.String(fname),
		})
	//log
	if err != nil {
		exitErrorf("Unable to download item %q, %v", fname, err)
	} else {
		log.Println("Sucessfully downloaded  ", numBytes, "byte file... attempting read...")
	}

	rows, err := csv.NewReader(inFile).ReadAll()
	//log
	if err != nil {
		log.Fatal(err)
		exitErrorf("Unable to open ", outFile, " %q, %v", err)
	} else {
		log.Println("Successfully read CSV file")
		log.Println("CSV READING CHECK: ", rows[0][0])
	}
	log.Println("Processing CSV file ... ")

	duplicates := appendCSV(rows)

	log.Println("Done. Deleted ", duplicates, " duplicate rows... \n writing to output file")
	log.Println("CSV READING CHECK: ", rows[0][0])

	//err = csv.NewWriter(outFile).WriteAll(rows)
	writer := csv.NewWriter(outFile)
	for _, row := range rows {
		err := writer.Write(row)
		if nil != err {
			log.Println("error writing row... err: ", err)
		}
	}

	if err != nil {
		log.Fatal(err)
		exitErrorf("Failed to write to output file.")
	}

	log.Println("Creating S3 uploader")
	uploader := s3manager.NewUploader(sess)

	outFileBody, err := ioutil.ReadFile(outFile.Name())
	if err != nil {
		exitErrorf("Failed to read \"%s\" => %v", outFile.Name(), err.Error())
	}

	log.Println("uploading output file to S3")
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bname),
		Key:    aws.String("outfile_" + fname),
		Body:   bytes.NewBuffer(outFileBody),
	})

	if err != nil {
		exitErrorf("Unable to upload %q to %q, %v", fname, "outfile_"+fname, err)
	} else {
		log.Println("Sucessfully uploaded " + outFile.Name() + "to S3 bucket:" + bname)
	}

	err = os.Remove(inFile.Name())
	if err != nil {
		log.Println("error removing input file \n error: ", err)
	}
	err = inFile.Close()
	if err != nil {
		log.Println("error closing input file \n error: ", err)
	}
	err = outFile.Close()
	if err != nil {
		log.Println("error closing output file \n error: ", err)
	}

}
func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	log.Println("System exiting due to error: " + msg)
	os.Exit(1)
}

/** Lambda Functions above */
func main() {
	lambda.Start(HandleRequest)
}

/** CSV Processing funcs below. */

func appendCSV(rows [][]string) int {
	rows[0] = append(rows[0], "Order Processing Time")
	rows[0] = append(rows[0], "Gross Margin")
	orderIDs := make(map[string]int) //keep track of existing orders
	duplicate := 0
	i := 1
	rowsToGo := len(rows)
	for ; i < rowsToGo; i++ {
		if strings.Compare(rows[i][1], "Mozambique") == 0 {
			fmt.Print("")
		}
		id := rows[i][6]
		_, isDuplicate := orderIDs[id] //check if row is a duplicate
		if isDuplicate {
			duplicate++
			rows = append(rows[:i], rows[i+1:]...) //if duplicate, remove row and move to next
			i--                                    //decrement i, to account for row shifting
			rowsToGo--
		} else {
			orderIDs[id] = i
			rows[i][4] = modifyPriority(rows[i][4])                  //change priority ; "H" -> "High" etc
			rows[i] = append(rows[i], calcOrderProcessTime(rows[i])) //calculate time between order and shipping
			rows[i] = append(rows[i], calcGrossMargin(rows[i]))      //calculate the gross margin profit/revenue
		}
	}
	return duplicate
}
func modifyPriority(priority string) string {
	switch priority {
	case "L":
		return "Low"
	case "M":
		return "Medium"
	case "H":
		return "High"
	case "C":
		return "Critical"
	}
	return priority
}
func calcGrossMargin(row []string) string {
	prof, err := StrToFloat(row[13])
	rev, err := StrToFloat(row[11])
	if err != nil {
		log.Fatal(err)
		return "err"
	} else {
		return fmt.Sprintf("%.3f", prof/rev)
	}
}
func StrToFloat(str string) (float64, error) {
	return strconv.ParseFloat(str, 64)
}
func calcOrderProcessTime(row []string) string {
	now.TimeFormats = append(now.TimeFormats, "2006 02 Jan 15:04")
	start, err := now.Parse(row[5])
	if err != nil {
		log.Fatal(err)
	}
	end, err := now.Parse(row[7])
	if err != nil {
		log.Fatal(err)
	}
	return strings.TrimRight(end.Sub(start).String(), "0m0s")
}

/** function for testing locally outside of lambda */
func testmain() {

	rows := [][]int{
		{0, 1, 10, 3},
		{4, 5, 6, 7},
		{8, 9, 10, 11},
		{4, 5, 10, 7},
		{2, 15, 1, 23},
	}

	var duplicates int

	orderIDs := make(map[int]int)
	for i := 0; i < len(rows); i++ {
		id := rows[i][2]
		_, duplicate := orderIDs[id]
		if duplicate {
			duplicates++
			rows = append(rows[:i], rows[i+1:]...)
		} else {
			orderIDs[id] = i
			rows[i] = append(rows[i], -1)
		}
	}
	fmt.Println(rows)
}
