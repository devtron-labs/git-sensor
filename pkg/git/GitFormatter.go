package git

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const _dl_ = "devtron_delimiter"

// GITFORMAT Refer git official doc for supported placeholders to add new fields
// Need to make sure the output does not break the json structure which is ensured
// by having the _dl_ delimiter which is later replaced by quotes
var GITFORMAT = "--pretty=format:{" +
	_dl_ + "commit" + _dl_ + ":" + _dl_ + "%H" + _dl_ + "," +
	_dl_ + "parent" + _dl_ + ":" + _dl_ + "%P" + _dl_ + "," +
	_dl_ + "refs" + _dl_ + ":" + _dl_ + "%D" + _dl_ + "," +
	_dl_ + "subject" + _dl_ + ":" + _dl_ + "%s" + _dl_ + "," +
	_dl_ + "body" + _dl_ + ":" + _dl_ + "%b" + _dl_ + "," +
	_dl_ + "author" + _dl_ +
	":{" +
	_dl_ + "name" + _dl_ + ":" + _dl_ + "%aN" + _dl_ + "," +
	_dl_ + "email" + _dl_ + ":" + _dl_ + "%aE" + _dl_ + "," +
	_dl_ + "date" + _dl_ + ":" + _dl_ + "%ad" + _dl_ +
	"}," +
	_dl_ + "commiter" + _dl_ +
	":{" +
	_dl_ + "name" + _dl_ + ":" + _dl_ + "%cN" + _dl_ + "," +
	_dl_ + "email" + _dl_ + ":" + _dl_ + "%cE" + _dl_ + "," +
	_dl_ + "date" + _dl_ + ":" + _dl_ + "%cd" + _dl_ +
	"}},"

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

func parseFormattedLogOutput(out string) ([]GitCommitFormat, error) {
	//remove the new line character which is after each terminal comma
	out = strings.ReplaceAll(out, "},\n", "},")

	// to escape the special characters like quotes and newline characters in the commit data
	logOut := strconv.Quote(out)

	//replace the delimiter with quotes to make it parsable json
	logOut = strings.ReplaceAll(logOut, _dl_, `"`)

	logOut = logOut[1 : len(logOut)-2]   // trim surround characters (surrounding quotes and trailing comma)
	logOut = fmt.Sprintf("[%s]", logOut) // Add []

	var gitCommitFormattedList []GitCommitFormat
	err := json.Unmarshal([]byte(logOut), &gitCommitFormattedList)
	if err != nil {
		return nil, err
	}
	return gitCommitFormattedList, nil
}

func (formattedCommit GitCommitFormat) transformToCommit() *Commit {
	return &Commit{
		Hash: &Hash{
			Long: formattedCommit.Commit,
		},
		Author: &Author{
			Name:  formattedCommit.Author.Name,
			Email: formattedCommit.Author.Email,
			Date:  formattedCommit.Author.Date,
		},
		Committer: &Committer{
			Name:  formattedCommit.Commiter.Name,
			Email: formattedCommit.Commiter.Email,
			Date:  formattedCommit.Commiter.Date,
		},
		Tag:     &Tag{},
		Tree:    &Tree{},
		Subject: formattedCommit.Subject,
		Body:    formattedCommit.Body,
	}
}
