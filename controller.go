package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
	"net/http"
	"os"
)

const CONTENT_TYPE = "application/json"
const TRANSFORM_ENDPOINT = "https://nkg8ojlm50.execute-api.us-east-1.amazonaws.com/Production/transform"
const LOAD_ENDPOINT = "https://nkg8ojlm50.execute-api.us-east-1.amazonaws.com/Production/load"

//const EXTRACTION_ENDPOINT = "";

type Requests struct {
	Bucket string `json:"bucketname"`
	File   string `json:"filename"`
	//Response Response `json:"response"`
}

type Responses struct {
	Success bool `json:"success"`
	//Transform return values
	Bucket  string `json:"bucketname"`
	FileIn  string `json:"input_filename"`
	FileOut string `json:"exported_filename"`
	Deleted int    `json:"deleted_duplicates"`
	//Load return values
	DB_file  string `json:"db_fileName"`
	DB_table string `json:"db_tableName"`
	//Extraction return values
	HTTPCODE int    `json:"httpcode"`
	DATA     string `json:"data"`
}

func HandleRequests(ctx context.Context, req Requests) (Responses, error) {

	log.Println("===CONTEXT===")
	log.Println(ctx)
	log.Println("=============")

	//var bucket string
	//if "" == req.Bucket {
	//	bucket = "tcss562.project.tsb"
	//} else {
	//	bucket = req.Bucket
	//}
	//inFileName := req.File;

	bucket := os.Getenv("BUCKET")
	fname := os.Getenv("FILE")
	DB := os.Getenv("DB_NAME")
	TABLE := os.Getenv("FILE")

	var res = Responses{}

	res.Bucket = bucket
	res.FileIn = fname

	log.Println("PERFORMING TRANSFORM")
	//Create json request message
	requestmessage := map[string]interface{}{
		"bucketname": bucket,
		"filename":   fname,
		"dbname":     DB,
		"tablename":  TABLE,
	}
	log.Println("request: ", requestmessage)
	jsonBytes, err := json.Marshal(requestmessage)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(jsonBytes)
	var jsonMsg = bytes.NewBuffer(jsonBytes)
	log.Println("jsonMsg:  ", jsonMsg)

	resp, err := http.Post(TRANSFORM_ENDPOINT, CONTENT_TYPE, jsonMsg)
	if err != nil {
		log.Fatal(err)
	}

	var result1 map[string]interface{}

	json.NewDecoder(resp.Body).Decode(&result1)

	log.Println(result1)

	// ============ FIRST SERVICE CALL ============= //

	if true != result1["success"] {
		log.Println("UNSUCCESSFUL FIRST CALL")
		//TODO: Terminate program

		return res, err

	} else {
		log.Println("SUCCESSFUL FIRST CALL")
		jsonBytes = nil
		resp = nil

		// ============ SECOND SERVICE CALL ============= //

		log.Println("Creating second message")
		requestmessage := map[string]interface{}{
			"bucketname": bucket,
			"filename":   fname,
			"dbname":     DB,
			"tablename":  TABLE,
		}
		log.Println("request: ", requestmessage)
		jsonBytes, err := json.Marshal(requestmessage)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(jsonBytes)
		var jsonMsg = bytes.NewBuffer(jsonBytes)
		log.Println("jsonMsg:  ", jsonMsg)

		log.Println("SECOND CALL MESSAGE:  ", jsonMsg)

		log.Println("finished marshalling message into json")
		log.Println(jsonMsg)

		resp, err := http.Post(LOAD_ENDPOINT, CONTENT_TYPE, jsonMsg)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Finished second call, checking results...")
		log.Println(resp)

		var result2 map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result2)

		log.Println(result2)
		del := result2["deleted_duplicates"]

		log.Println("DELETED:", del)

		if false == result2["success"] {
			log.Println("UNSUCCESSFUL SECOND CALL")
			//TODO: Terminate program

			return res, err

		} else {
			log.Println("SUCCESSFUL SECOND CALL")

		}

	}

	log.Println(resp)

	//return Responses{Success: true, Bucket:bucket}, err

	//	return Responses{Success: false}, nil
	return res, err
}

func main() {
	lambda.Start(HandleRequests)
}
