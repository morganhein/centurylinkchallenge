centurylinkchallenge

To run this, either download the main.exe on a windows computer and run like any other program,
or download this rep in go using "go get github.com/morganhein/centurylinkchallenge",  
navigate to the clc folder, and type "go run main.go".

The server uses port 8080, and expects JSON in the following format:

{
    "name": "example",
    "cpu": 16.23,
    "mem": 33.2342,
    "time": "2016-04-24T18:51:00-07:00"
}

Where name is a string, cpu, mem is a double/float64, and time is a datetime formatted in RFC3339.
