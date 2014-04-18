### Wapa Project

Wapa is a tool to help you manage command job execution across servers via a SQL database

### Installation

* Get the binary and source of wapa by cloning this repo

```bash
git clone https://github.com/jiananlu/wapa.git
```

* Make sure ```wapa/bin``` is in your ```PATH```

* Create a file named: ```~/.waparc``` with following content:

```
{
    "DBuser": "username",
    "DBpass": "password",
    "DBhost": "sql.yourhost.com",
    "DBport": "3306",
    "DBname": "dbname",
    "Encryption_key": "<some 32 bits encryption key>"
}
```

### Usage

The help message of wapa:

```
Usage:
---------------
wapa --list <all|wait|run|done>
    to list jobs that are currently in database

wapa --add <command>
    to add <command> as a new job to database

wapa --add-file <file>
    every line in the file will be added as a new job into database

wapa --clean
    to clean all completed jobs
    
wapa --clean --all
    to clean all jobs

wapa --go
    start the scheduler to fetch jobs from database and work on them
```