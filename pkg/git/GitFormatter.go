package git

import "time"

var GITFORMAT = `--pretty=format:{"commit":"%H","parent":"%P","refs":"%D","subject":"%s","body":"%b","author":{"name":"%aN","email":"%aE","date":"%ad"},"commiter":{"name":"%cN","email":"%cE","date":"%cd"}},`

type GitPerson struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}
type GitCommitFormat struct {
	Commit   string    `json:"commit"`
	Parent   string    `json:"parent"`
	Refs     string    `json:"refs"`
	Subject  string    `json:"subject"`
	Commiter GitPerson `json:"commiter"`
	Author   GitPerson `json:"author"`
	Body     string    `json:"body"`
}
