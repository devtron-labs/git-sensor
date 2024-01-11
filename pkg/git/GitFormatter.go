package git

import "time"

// var GITFORMAT = `--pretty=format:{"commit":"%H","parent":"%P","refs":"%D","subject":"%s","body":"%b","author":{"name":"%aN","email":"%aE","date":"%ad"},"commiter":{"name":"%cN","email":"%cE","date":"%cd"}},`
var GITFORMAT = "--pretty=format:{devtron_delimitercommitdevtron_delimiter:devtron_delimiter%Hdevtron_delimiter,devtron_delimiterparentdevtron_delimiter:devtron_delimiter%Pdevtron_delimiter,devtron_delimiterrefsdevtron_delimiter:devtron_delimiter%Ddevtron_delimiter,devtron_delimitersubjectdevtron_delimiter:devtron_delimiter%sdevtron_delimiter,devtron_delimiterbodydevtron_delimiter:devtron_delimiter%bdevtron_delimiter,devtron_delimiterauthordevtron_delimiter:{devtron_delimiternamedevtron_delimiter:devtron_delimiter%aNdevtron_delimiter,devtron_delimiteremaildevtron_delimiter:devtron_delimiter%aEdevtron_delimiter,devtron_delimiterdatedevtron_delimiter:devtron_delimiter%addevtron_delimiter},devtron_delimitercommiterdevtron_delimiter:{devtron_delimiternamedevtron_delimiter:devtron_delimiter%cNdevtron_delimiter,devtron_delimiteremaildevtron_delimiter:devtron_delimiter%cEdevtron_delimiter,devtron_delimiterdatedevtron_delimiter:devtron_delimiter%cddevtron_delimiter}},"

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
