# csv_lambda_func

This project is intended as part of a series of microservices intended to be deployed to AWS Lambda. 

## To Use:
1. Compile the main go file: ``` go build -o main```
2. zip and upload the main executable to your lambda function
3. Test the function using this JSON as a configured test event. This same json format is also the expected input.
``` JSON
{
  "bucketname": "tcss562.project.tsb",
  "filename": "1000 Sales Records.csv"
}
```
* **bucketname** -> The S3 bucket name that the input files are stored in.
* **filename**   -> The CSV file to process.

The output will be loaded into the same bucket, as ```outfile_<original file name>.csv```
