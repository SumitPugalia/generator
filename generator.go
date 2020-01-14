package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var Service string
var ServiceAbbreviation string
var Model string
var LowerCaseModel string
var Attributes map[string]string

func main() {
	filename := os.Args[1]
	read_file(filename)
	LowerCaseModel = strings.ToLower(Model)
	r := strings.Split(Service, "-")
	for _, value := range r {
		ServiceAbbreviation += string(value[0])
	}
	fmt.Println(ServiceAbbreviation)

	// Create Service Structure
	dir := Service + "/" + "domain/" + "entity"
	create_dir(dir)
	create_file(dir, LowerCaseModel+".go", entity_model_data())
	create_file(Service+"/"+"domain", "service.go", service_interface_data())

	// // Create Core -> repos Structure
	// core := Service + "/" + "core"
	// create_dir(core)
	// create_dir(core + "/" + "repos")
	// create_file(core+"/"+"repos", LowerCaseModel+".go", repos_models_data())
	// create_file(core, "core.go", core_core_data())

}

func create_dir(dirName string) error {
	err := os.MkdirAll(dirName, 0777)
	if err == nil || os.IsExist(err) {
		return nil
	} else {
		log.Fatal(err)
		return err
	}
}

func create_file(dirPath string, name string, data []byte) {
	dst, err := os.Create(filepath.Join(dirPath, filepath.Base(name)))
	if err != nil {
		log.Fatal(err)
	}

	n, err := dst.Write(data)
	fmt.Println(n)
	defer dst.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func read_file(name string) {
	file, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	m := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()

		matched_1, _ := regexp.MatchString(`ServiceName =`, line)
		if matched_1 {
			re := regexp.MustCompile(`ServiceName = `)
			Service = re.ReplaceAllString(line, "")
		}

		matched_2, _ := regexp.MatchString(`ModelName =`, line)
		if matched_2 {
			re := regexp.MustCompile(`ModelName = `)
			Model = re.ReplaceAllString(line, "")
		}

		matched_3, _ := regexp.MatchString(`Attributes = {`, line)
		matched_4, _ := regexp.MatchString(`}`, line)

		if !(matched_1 || matched_2 || matched_3 || matched_4) {
			attribute := strings.Trim(line, " ")
			res1 := strings.Split(attribute, "=")
			m[res1[0]] = res1[1]
		}
	}

	Attributes = m

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func entity_model_data() []byte {
	b := new(bytes.Buffer)
	for key, value := range Attributes {
		fmt.Fprintf(b, "\t%s %s\n", key, value)
	}

	data := "package entity\n\ntype " + Model + " struct {\n" + b.String() + "}"
	return []byte(data)
}

func service_interface_data() []byte {
	b := new(bytes.Buffer)
	for key, value := range Attributes {
		if key != "Id" {
			fmt.Fprintf(b, "\t%s %s\n", key, value)
		}
	}

	data :=
		"package domain\n\n" +
			"import \"" + Service + "/entity\"\n\n" +
			"type Service interface {\n" +
			service_functions("entity.") +
			"}\n\n" +
			"//Please remove the attribute that is not required for create or update params\n\n" +
			"type Create" + Model + "Params struct {\n" + b.String() + "}\n\n" +
			"type Update" + Model + "Params struct {\n" + b.String() + "}"

	return []byte(data)
}

func repos_models_data() []byte {
	data := "package repos\n\n" +
		"import " + ServiceAbbreviation + " \"" + Service + "/service\"\n\n" +
		"type " + Model + "Repo " + "interface {\n" +
		service_functions(ServiceAbbreviation+".") + "}"

	return []byte(data)
}

func core_core_data() []byte {
	data := "// FIXME: Figure out better naming convention for the package (service implementation) \n" +
		"// Contains implementation of \"Service\" interface \n" +
		"package core \n\n" +
		"import (\n" +
		"\t\"fmt\"\n" +
		"\t\"" + Service + "/core/db\"\n" +
		"\trepos2 \"" + Service + "/core/db/repos\"\n" +
		"\t\"" + Service + "/core/repos\"\n" +
		"\t\"" + Service + "/service\"\n" +
		")\n\n" +
		"type coreService struct {\n" +
		"\t" + LowerCaseModel + "Repo" + " " + "repos." + Model + "Repo\n" +
		"}\n\n" +
		make_service() + "\n" +
		make_core_functions_data()

	return []byte(data)
}

func service_functions(prefix string) string {
	list := "\tList" + Model + "s() ([]" + prefix + Model + ", error)\n"
	get := "\tGet" + Model + "(id string) (" + prefix + Model + ", error)\n"
	create := "\tCreate" + Model + "(" + LowerCaseModel + " " + prefix + Model + ") (" + prefix + Model + ", error)\n"
	update := "\tUpdate" + Model + "(" + LowerCaseModel + " " + prefix + Model + ") (" + prefix + Model + ", error)\n"
	del := "\tDelete" + Model + "(id string) (interface{}, error)\n"

	return list + get + create + update + del
}

func make_service() string {
	data := "func MakeService() service.Service {\n" +
		"\trDB, err := db.GetPostgresDB()\n" +
		"\tif err != nil {\n\t\tfmt.Println(\"Error getting Read DB\")\n\t}\n" +
		"\twDB, err := db.GetPostgresDB()\n" +
		"\tif err != nil {\n\t\tfmt.Println(\"Error getting Write DB\")\n\t}\n" +
		"\tbr := repos2.Make" + Model + "Repo(wDB, rDB)\n" +
		"\tmr := repos2.MakeModelRepo(wDB, rDB)\n" +
		"\treturn coreService{br, mr}\n" + "}\n"

	return data
}

func make_core_functions_data() string {
	data :=
		"func (s coreService) List" + Model + "s() ([]service." + Model + ", error) {\n" +
			"\treturn s." + LowerCaseModel + "Repo.List" + Model + "s()\n" +
			"}\n\n" +

			"func (s coreService) Get" + Model + "(id int64) (service." + Model + ", error) {\n" +
			"\treturn s." + LowerCaseModel + "Repo.Get" + Model + "(id)\n" +
			"}\n\n" +

			"func (s coreService) Create" + Model + "(" + LowerCaseModel + " service." + Model + ") (service." + Model + ", error) {\n" +
			"\treturn s." + LowerCaseModel + "Repo.Create" + Model + "(" + LowerCaseModel + ")\n" +
			"}\n\n" +

			"func (s coreService) Update" + Model + "(" + LowerCaseModel + " service." + Model + ") (service." + Model + ", error) {\n" +
			"\treturn s." + LowerCaseModel + "Repo.Update" + Model + "(" + LowerCaseModel + ")\n" +
			"}\n\n" +

			"func (s coreService) Delete" + Model + "(id int64) (interface{}, error) {\n" +
			"\treturn s." + LowerCaseModel + "Repo.Delete" + Model + "(id)\n" +
			"}\n\n"
	return data
}
