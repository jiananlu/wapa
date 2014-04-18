package main

import (
    "fmt"
    "database/sql"
    "flag"
    _ "github.com/go-sql-driver/mysql"
    "wapa/encrypt"
    "wapa/config"
    "time"
    "os/exec"
    "io/ioutil"
    "strings"
    "os/user"
)

var (
    flagList string
    flagAdd string
    flagAddFile string
    flagClean bool
    flagAll bool
    flagGo bool
)

func usage() {
    fmt.Println(`
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
`)
}

func init() {
    flag.StringVar(&flagList, "list", "", "list <wait|run|done|all> jobs.")
    flag.StringVar(&flagAdd, "add", "", "add <command> as a new job.")
    flag.StringVar(&flagAddFile, "add-file", "", "add all lines in file as commands")
    flag.BoolVar(&flagClean, "clean", false, "clean all jobs marked as done.")
    flag.BoolVar(&flagAll, "all", false, "clean all jobs.")
    flag.BoolVar(&flagGo, "go", false, "start to work out all jobs in database.")
    flag.Usage = usage
}

type MyDB struct {
    user string
    pass string
    host string
    port string
    dbname string
    db *sql.DB
}

func (mydb *MyDB) setup() error {
    db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", mydb.user, mydb.pass, mydb.host, mydb.port, mydb.dbname))
    if err != nil {
        return err // something wrong happened during setup
    }
    mydb.db = db
    if err := mydb.db.Ping(); err != nil {
        return err // can't really ping the DB
    }
    return nil
}

type Job struct {
    hash string
    flag int
    cmd string
    status string
}

func (job *Job) decode() {
    if job.hash == "" {
        return
    }
    c, err := encrypt.NewCrypto()
    if err != nil {
        panic(err)
    }
    cmd, err := c.Decrypt(job.hash)
    if err != nil {
        panic(err)
    }
    job.cmd = cmd
    switch job.flag {
    case 0:
        job.status = "wait"
    case 1:
        job.status = "*run"
    case 2:
        job.status = "done"
    default:
        job.status = "eror"
    }
}

func (job *Job) encode() {
    if job.cmd == "" {
        return // no command to encode
    }
    c, err := encrypt.NewCrypto()
    if err != nil {
        panic(err)
    }
    hash, err := c.Encrypt(job.cmd)
    if err != nil {
        panic(err)
    }
    job.hash = hash
}

func (job *Job) toS(compact bool) string {
    var cmd string
    if compact && len(job.cmd) > 100 {
        cmd = fmt.Sprintf("%s...", job.cmd[:100])
    } else {
        cmd = job.cmd
    }
    return fmt.Sprintf("%s -> %s", job.status, cmd)
}

type JobManager struct {
    mydb *MyDB
    jobs []*Job
}

func (jm *JobManager) initDB() {
    u, err := user.Current()
    if err != nil {
        panic(err)
    }
    myConfig, err := config.NewConfig(fmt.Sprintf("%s/.waparc", u.HomeDir))
    if err != nil {
        panic(err)
    }
    user := myConfig.DBuser
    pass := myConfig.DBpass
    host := myConfig.DBhost
    port := myConfig.DBport
    dbname := myConfig.DBname
    
    if jm.mydb != nil && jm.mydb.db != nil {
        // all set nothing need to be don
        return
    }
  
    jm.mydb = &MyDB{user, pass, host, port, dbname, nil}
    err = jm.mydb.setup()
    if err != nil {
        panic(err)
    }
    return
}

func (jm *JobManager) getDB() *sql.DB {
    if jm.mydb == nil || jm.mydb.db == nil {
        // db handle not init yet let's do it now
        jm.initDB()
    }
    return jm.mydb.db
}

func (jm *JobManager) FetchJobs() {
    rows, err := jm.getDB().Query("select * from jobs")
    if err != nil {
        panic(err)
    }
    var hash string
    var flag int  
    // reset the jobs list
    jm.jobs = jm.jobs[:0]
    for rows.Next() {
        if err := rows.Scan(&hash, &flag); err != nil {
            return
        }
        job := &Job{hash: hash, flag: flag}
        job.decode()
        jm.jobs = append(jm.jobs, job)
    }
    return
}

func (jm *JobManager) PrintJobs(jobFlag int) {
    // update the jobs list in memory
    jm.FetchJobs()
    // now let's print all jobs
    for _, job := range jm.jobs {
        if jobFlag == -1 {
            fmt.Println(job.toS(true))
        } else {
            if jobFlag == job.flag {
                fmt.Println(job.toS(true))
            }
        }        
    }
}

func (jm *JobManager) AddJob(cmd string) {
    if len(cmd) == 0 {
        panic("invalid command")
    }
    job := Job{cmd: cmd}
    job.encode()
    _, err := jm.getDB().Exec("insert into jobs (hash, flag) values (?, ?)", job.hash, 0)
    if err != nil {
        panic(err)
    }
}

func (jm *JobManager) AddFile(fileName string) {
    contents, err := ioutil.ReadFile(fileName)
    if err != nil {
        panic(err)
    }
    lines := strings.Split(string(contents), "\n")
    cmds := []string{"insert into jobs (hash, flag) values"}
    count := 0
    for _, line := range lines {
        if line != "" {
            job := Job{cmd: line}
            job.encode()
            cmds = append(cmds, fmt.Sprintf("('%s', 0),", job.hash))
            count++
        }
    }
    sqlCmd := strings.Join(cmds, " ")
    // get ride of the last , if there is
    if strings.HasSuffix(sqlCmd, ",") {
        sqlCmd = sqlCmd[:len(sqlCmd) - 1]
    }
    _, err = jm.getDB().Exec(sqlCmd)
    if err != nil {
        panic(err)
    }
    fmt.Printf("%d jobs added.", count)
}

func (jm *JobManager) CleanJobs(all bool) {
    var sqlCmd string
    if all {
        // you want me to delete everything? okey dokey
        sqlCmd = "delete from jobs"
    } else {
        // only clean up all completed jobs
        sqlCmd = "delete from jobs where flag=2"
    }
    _, err := jm.getDB().Exec(sqlCmd)
    if err != nil {
        panic(err)
    }
}

func (jm *JobManager) updateJobFlag(job *Job, flag int) {
    if flag != 0 && flag != 1 && flag != 2 {
        panic("invalid flag to update to")
    }
    if job.hash == "" {
        job.encode()
    }
    _, err := jm.getDB().Exec("update jobs set flag=? where hash=?", flag, job.hash)
    if err != nil {
        panic(err)
    }
}

func (jm *JobManager) Go() {
    for {
        fmt.Println("fetching next job...")
        job, shouldWait := jm.GetNextJob()
        if shouldWait {
            fmt.Println("no job in the queue, wait for 5 minutes before next poll")
            time.Sleep(time.Minute * 5)
        }
        fmt.Println("makring the job as running in db")
        jm.updateJobFlag(job, 1)
        err := jm.runCmd(job.cmd)
        if err != nil {
            fmt.Println("something wrong with this command, restore it to <wait>")
            jm.updateJobFlag(job, 0)
            panic(err)
        }
        fmt.Println("makring the job as done in db")
        jm.updateJobFlag(job, 2)
    }
}

func (jm *JobManager) runCmd(cmdStr string) error {
    // cmd needs to be name + args
    cmds := strings.Fields(cmdStr)
    cmdName := cmds[0]
    arg := strings.Join(cmds[1:], " ")
    out, err := exec.Command(cmdName, arg).Output()
    outStr := string(out)
    if err != nil {
        fmt.Println(outStr)
        return err
    }
    return nil
}

func (jm *JobManager) GetNextJob() (*Job, bool) {
    rows, err := jm.getDB().Query("select * from jobs where flag=0 limit 1")
    if err != nil {
        panic(err)
    }
    var hash string
    var flag int
    if rows.Next() {
        if err := rows.Scan(&hash, &flag); err != nil {
            panic(err)
        }
        job := &Job{hash: hash, flag: flag}
        job.decode()
        return job, false
    } else {
        return &Job{}, true
    }
}

func main() {
    flag.Parse()    
    jm := &JobManager{}

    switch flagList {
    case "all":
        jm.PrintJobs(-1)
        return
    case "wait":
        jm.PrintJobs(0)
        return        
    case "run":
        jm.PrintJobs(1)
        return        
    case "done":
        jm.PrintJobs(2)
        return        
    }
    
    if flagAdd != "" {
        jm.AddJob(flagAdd)
        return
    }
    
    if flagAddFile != "" {
        jm.AddFile(flagAddFile)
        return
    }
    
    if flagClean {
        jm.CleanJobs(flagAll)
        return
    }
    
    if flagGo {
        // this is in worker mode
        jm.Go()
    }
    
    usage()
}