package coverage

import (
	"fmt"
	"html/template"
	"os"
)

/*
<!DOCTYPE html>
<html>

	<body>
	    <p> {{.Code}} </p>
	</body>

</html>
*/

type Page struct {
	Code string
}

// Uses an html template to generate the coverage report for a file
func writeCoverageFile(path string, name string, coverageData []byte) {
	tmpl, err := template.New("layout.html").ParseFiles("layout.html")
	if err != nil {
		// TODO error handling
		fmt.Println("error parsing template", err)
	}

	file, err := os.Create("coverage.html")
	if err != nil {
		// TODO error handling
		fmt.Println("error creating file", err)
	}

	defer file.Close()

	err = tmpl.Execute(file, coverageData)
	if err != nil {
		// TODO error handling
		fmt.Println("error executing template", err)
	}
}
