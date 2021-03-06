/**
* Tanner Brown
* AWS Lambda function to retrieve data from a CVS file saved in S3, modify it,
* and store it as a new CSV file to be loaded into a database
*/
package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	lambda2 "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/jinzhu/now"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

type Request struct {
	Bucket string `json:"bucketname"`
	File   string `json:"filename"`
	ID     string `json:"transactionid"`
}

type Response struct {
	Success bool   `json:"success"`
	Bucket  string `json:"bucketname"`
	FileIn  string `json:"input_filename"`
	FileOut string `json:"exported_filename"`
	Error   string `json:"error"`
	Deleted int    `json:"deleted_duplicates"`
	ID      string `json:"transactionid"`
	RunTime int    `json:runtime`
}


func HandleRequest(ctx context.Context, req Request) (Response, error) {
	var uuid = ""
	fname := req.File
	bname := req.Bucket
	secondFuncName := os.Getenv("FUNCTION_NAME")
	success := false

	lc, _ := lambdacontext.FromContext(ctx)
	if req.ID != "" {
		uuid = req.ID
	} else {
		uuid = lc.AwsRequestID
	}

	//var id = ctx.Value("AwsRequestID")
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

	duplicates := appendCSV(rows) //make changes and note number of duplicate rows

	cropRows := len(rows) - duplicates //count and crop extra rows due to duplicates
	rows = append(rows[:cropRows])

	log.Println("Done. Deleted ", duplicates, " duplicate rows... \n writing to output file")
	log.Println("CSV READING CHECK: ", rows[0][0])

	writer := csv.NewWriter(outFile)
	writer.WriteAll(rows)

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
		Key:    aws.String("/transformed/" + fname),
		Body:   bytes.NewBuffer(outFileBody),
	})

	if err != nil {
		exitErrorf("Unable to upload %q to %q, %v", fname, "outfile_"+fname, err)
	} else {
		log.Println("Sucessfully uploaded " + "/transformed/" + fname + "to S3 bucket:" + bname)
		success = true
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
	errStr := "No Errors"
	if err != nil {
		errStr = err.Error()
	}

	if len(secondFuncName) > 0 {
		log.Println("Calling function: ", secondFuncName)

		requestmessage := map[string]interface{}{
			"filename":      fname,
			"transactionid": uuid,
		}
		jsonBytes, err := json.Marshal(requestmessage)
		if err != nil {
			log.Fatal(err)
		}

		lambdaFunc := lambda2.New(sess)
		input := &lambda2.InvokeInput{
			FunctionName: aws.String(secondFuncName),
			Payload:      jsonBytes,
		}

		result, err := lambdaFunc.Invoke(input)
		if err != nil {
			log.Fatal(err)
		} else {
			log.Println("second func result: ", result)
		}
	}

	return Response{Success: success, Bucket: bname, FileIn: fname, FileOut: "/transformed/" + fname, Error: errStr, Deleted: duplicates, ID: uuid}, err
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
		//if strings.Compare(rows[i][1], "Mozambique") == 0 {
		//	fmt.Print("")
		//}
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

func testMainWithCSV() {
	fname := "100SalesRecords.csv"
	inFile, err := os.Open(fname)
	if nil != err {
		log.Fatal(err)
		exitErrorf("Unable to open ", inFile, " %q, %v", err)
	} else {
		log.Println("created infile " + inFile.Name())
	}
	outFile, err := os.Create("outfile.csv")
	if nil != err {
		log.Fatal(err)
		exitErrorf("Unable to open ", outFile, " %q, %v", err)
	} else {
		log.Println("created outfile " + outFile.Name())
	}

	defer inFile.Close()
	defer outFile.Close()

	rows, err := csv.NewReader(inFile).ReadAll()
	duplicates := appendCSV(rows)

	cropRows := len(rows) - duplicates
	rows = append(rows[:cropRows])

	fmt.Println("DUPLICATES: ", duplicates)
	writer := csv.NewWriter(outFile)
	writer.WriteAll(rows)

}
