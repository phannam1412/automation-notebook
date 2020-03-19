package repository

import (
	"common"
	"database/sql"
	"fmt"
)

func SaveJobToDb(db *sql.DB, res common.JobResult) {
	result := "passed"
	if res.Result == false {
		result = "failed"
	}
	fmt.Printf("saving to db... job code: %s, status: %s, execution time: %s\n", res.Job.Code, result, res.ExecutionTime)
	if !JobExists(db, res.Job.Content, res.Job.Type) {
		_, err := db.Exec(`INSERT INTO job(content, type, code, output, status) VALUES(?, ?, ?, ?, ?)`, res.Job.Content, res.Job.Type, res.Job.Code, res.Output, result)
		common.PanicOnError(err)
		return
	}
	_, err := db.Exec(`UPDATE job SET output = ?, status = ?, content = ? WHERE code = ? AND type = ?`, res.Output, result, res.Job.Content, res.Job.Code, res.Job.Type)
	common.PanicOnError(err)
}

func JobExists(db *sql.DB, content string, jobType string) bool {
	rows, err := common.FetchAll(db, `SELECT 1 FROM job WHERE content = ? AND type = ?`, content, jobType)
	common.PanicOnError(err)
	return len(rows) > 0
}

func FilterByJobExistsAndPassed(db *sql.DB, jobs []common.Job) []common.Job {
	codes, err := common.FetchAll(db, `SELECT code FROM job WHERE status = 'passed'`)
	common.PanicOnError(err)
	passed := map[string]bool{}
	for _, v := range codes {
		if v, ok := v["code"].(string); ok {
			passed[v] = true
		}
	}
	var filtered []common.Job
	for _, job := range jobs {
		if _, ok := passed[job.Code]; ok {
			continue
		}
		filtered = append(filtered, job)
	}
	return filtered
}

func Resolver(resultChan chan common.JobResult) {
	db, err := sql.Open("mysql", "root:root@tcp(localhost:3307)/help_db")
	common.PanicOnError(err)
	for {
		res, more := <-resultChan
		if !more {
			break
		}
		SaveJobToDb(db, res)
	}
}