# Microservice controller:
[controller](https://files.slack.com/files-pri/TDCHQBP63-FEKGWMWF9/image.png)


# csv_lambda_func

This project is intended as part of a series of microservices intended to be deployed to AWS Lambda. 

## To Use:
1. Compile the main go file: ``` go build -o main```
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
  "input_filename": "100SalesRecords.csv",
  "exported_filename": "outfile_100SalesRecords.csv",
  "error": "No Errors",
  "deleted_duplicates": 3
}
```
