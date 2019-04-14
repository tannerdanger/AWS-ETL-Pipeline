/**
* Tanner Brown
* This file calls all 3 components of the ETL pipeline to test using one
* single controller to call all 3 ETL services
*/

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"log"
	"net/http"
	"os"
	"time"
)

const CONTENT_TYPE = "application/json"
const TRANSFORM_ENDPOINT = "https://nkg8ojlm50.execute-api.us-east-1.amazonaws.com/Production/transform"
const LOAD_ENDPOINT = "https://nkg8ojlm50.execute-api.us-east-1.amazonaws.com/Production/load"
const EXTRACTION_ENDPOINT = "https://nkg8ojlm50.execute-api.us-east-1.amazonaws.com/Production/extract"

//const EXTRACTION_ENDPOINT = "";

type Requests struct {
	Bucket string `json:"bucketname"`
	File   string `json:"filename"`
	//Response Response `json:"response"`
}

type Responses struct {
	Success bool   `json:"success"`
	ID      string `json:"transactionid"`
	//Transform return values
	Bucket  string `json:"bucketname"`
	FileIn  string `json:"input_filename"`
	FileOut string `json:"exported_filename"`
	Deleted int    `json:"deleted_duplicates"`
	//Load return values
	DB_file  string `json:"db_fileName"`
	DB_table string `json:"db_tableName"`
	//Extraction return values
	RESP1     map[string]interface{} `json:"response1"`
	RESP2     map[string]interface{} `json:"response2"`
	RESP3     map[string]interface{} `json:"response3"`
	Timer1    time.Duration          `json:"runtime1"`
	Timer2    time.Duration          `json:"runtime2"`
	Timer3    time.Duration          `json:"runtime3"`
	TotalTime time.Duration          `json:"runtimeTotal"`
}

func HandleRequests(ctx context.Context, req Requests) (Responses, error) {
	NetClient := &http.Client{Timeout: time.Minute * 15}
	NetClient.Timeout = time.Minute * 15

	var uuid = ""
	log.Println("===CONTEXT===")
	lc, _ := lambdacontext.FromContext(ctx)
	log.Println(lc)
	uuid = lc.AwsRequestID
	log.Println("=============")

	bucket := os.Getenv("BUCKET")
	fname := req.File
	//DB := os.Getenv("DB_NAME")
	//TABLE := os.Getenv("FILE")

	var res = Responses{}

	res.Bucket = bucket
	res.FileIn = fname

	log.Println("PERFORMING TRANSFORM")
	//Create json request message
	requestmessage := map[string]interface{}{
		"bucketname": bucket,
		"filename":   fname,
		"requestid":  uuid,
	}

	jsonBytes, err := json.Marshal(requestmessage)

	if err != nil {
		log.Fatal(err)
	}

	var jsonMsg = bytes.NewBuffer(jsonBytes)
	log.Println("jsonMsg:  ", jsonMsg)

	start := time.Now()
	resp, err := NetClient.Post(TRANSFORM_ENDPOINT, CONTENT_TYPE, jsonMsg)
	if err != nil {
		log.Fatal(err)
	}
	time1 := time.Since(start)

	var result1 map[string]interface{}

	json.NewDecoder(resp.Body).Decode(&result1)

	log.Println(result1)

	// ============ FIRST SERVICE CALL ============= //

	if true != result1["success"] {
		log.Println("UNSUCCESSFUL TRANSFORM CALL")
		//TODO: Terminate program
		return res, err

	} else {
		log.Println("SUCCESSFUL TRANSFORM CALL")
		res.RESP1 = result1
		res.Timer1 = time1

		jsonBytes = nil
		resp = nil

		// ============ SECOND SERVICE CALL ============= //

		log.Println("Creating load message")
		requestmessage := map[string]interface{}{
			"filename":      fname,
			"transactionid": uuid,
		}

		jsonBytes, err := json.Marshal(requestmessage)
		if err != nil {
			log.Fatal(err)
		}

		var jsonMsg = bytes.NewBuffer(jsonBytes)

		log.Println("finished marshalling message into json")
		log.Println("LOAD MESSAGE:  ", jsonMsg)

		srvicetimer := time.Now()
		resp, err := NetClient.Post(LOAD_ENDPOINT, CONTENT_TYPE, jsonMsg)
		if err != nil {
			log.Fatal(err)
		}
		time2 := time.Since(srvicetimer)
		log.Println("Finished load call, checking results...")
		log.Println(resp)

		var result2 map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result2)

		log.Println(result2)

		if false == result2["success"] {
			log.Println("UNSUCCESSFUL SECOND CALL")
			//TODO: Terminate program

			return res, err

		} else {
			log.Println("SUCCESSFUL LOAD CALL")
			jsonBytes = nil
			resp = nil
			res.Timer2 = time2
			res.RESP2 = result2

			// ============ SECOND SERVICE CALL ============= //

			log.Println("Creating EXTRACTION message")
			// LOAD
			requestmessage := map[string]interface{}{
				"bucketname":    bucket,
				"dbname":        result2["dbname"],
				"tablename":     result2["tablename"],
				"transactionid": uuid,
			}

			jsonBytes, err := json.Marshal(requestmessage)
			if err != nil {
				log.Fatal(err)
			}
			var jsonMsg = bytes.NewBuffer(jsonBytes)
			log.Println("finished marshalling message into json")
			log.Println("EXTRACTION CALL MESSAGE:  ", jsonMsg)

			srvicetimer := time.Now()
			resp, err := NetClient.Post(EXTRACTION_ENDPOINT, CONTENT_TYPE, jsonMsg)
			if err != nil {
				log.Fatal(err)
			}
			time3 := time.Since(srvicetimer)

			log.Println("Finished EXTRACTION call, checking results...")
			log.Println(resp)

			var result3 map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result3)

			log.Println(result3)

			if false == result3["success"] {
				log.Println("UNSUCCESSFUL EXTRACTION CALL")
				//TODO: Terminate program
				res.Success = false
				return res, err
			} else {
				totaltime := time.Since(start)
				res.TotalTime = totaltime
				res.Timer3 = time3
				res.Success = true
				res.RESP3 = result3
			}

		}

	}

	//return Responses{Success: true, Bucket:bucket}, err

	//	return Responses{Success: false}, nil
	return res, err
}

func main() {

	lambda.Start(HandleRequests)
}
