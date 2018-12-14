# csv_lambda_func

This project is intended as part of a series of microservices intended to be deployed to AWS Lambda. It includes a microservice for manipulating and appending a CSV file, and a controller for calling all three microservices.

## To Use CSV microservice:
1. Upload the included main.zip file to lambda (and skip step 2) OR Compile the main go file: ``` go build -o main```
2. zip and upload the main executable to your lambda function
3. Test the function using this JSON as a configured test event. This same json format is also the expected input.
```JSON
{
  "bucketname": "tcss562.project.tsb",
  "filename": "1000 Sales Records.csv"
}
```
Or use callservice.sh with the bucket name as param 1 and filename as param 2. 
```Bash
./callservice.sh tcss562.project.tsb 1000SalesRecords.csv
```

* **bucketname** -> The S3 bucket name that the input files are stored in.
* **filename**   -> The CSV file to process.

## Sample Output:
```JSON
{
  "success": true,
  "bucketname": "tcss562.project.tsb",
  "input_filename": "1000SalesRecords.csv",
  "exported_filename": "/transformed/1000SalesRecords.csv",
  "error": "No Errors",
  "deleted_duplicates": 0,
  "transactionid": "9f72637b-fd68-11e8-9f2c-a7be6afb8474"
}
```
## To Use Controller microservice:
1. Modify the api endpoints in the controller.go file
2. zip and upload the main executable to your lambda function
3. Test the function using this JSON as a configured test event. This same json format is also the expected input.
