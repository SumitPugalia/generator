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
	"unicode"
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

	// Domain Structure
	domain_dir := Service + "/" + "domain/" + "entity"
	create_dir(domain_dir)
	create_file(domain_dir, LowerCaseModel+".go", entity_model_data())
	create_file(Service+"/"+"domain", "service.go", service_interface_data())

	// Service structure
	service_dir := Service + "/" + "service"
	create_dir(service_dir)
	create_file(service_dir, "service.go", service_implementation())

	// Endpoint structure
	endpoint_dir := Service + "/" + "endpoint"
	create_dir(endpoint_dir)
	create_file(endpoint_dir, "decoder.go", decoder_data())
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

func service_implementation() []byte {
	data :=
		"package service \n\n" +
			"import (\n" +
			"\t\"" + Service + "/domain\"\n" +
			"\t\"" + Service + "/domain/entity\"\n" +
			"\t\"" + Service + "/repository\"\n" +
			"\t\"" + Service + "/repository/impl/postgresql\"\n" +
			")\n\n" +
			"type ServiceImpl struct {\n" +
			"\t" + LowerCaseModel + "Repo" + " " + "repository" + Model + "Repo\n" +
			"}\n\n" +
			"func MakeServiceImpl() ServiceImpl {\n" +
			"\t" + LowerCaseModel + "Repo := postgresql.MakePostgres" + Model + "Repo()\n" +
			"\treturn ServiceImpl{" + LowerCaseModel + "Repo: &" + LowerCaseModel + "Repo}\n" +
			"}\n\n" +
			service_implementation_functions()

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

func decoder_data() []byte {
	b := new(bytes.Buffer)
	for key, value := range Attributes {
		fmt.Fprintf(b, "\t%s %s %s\n", key, value, "`json:\""+LowerInitial(key)+"\"`")
	}

	data :=
		"package endpoint \n\n" +
			"import (\n" +
			"\t\"context\"\n" +
			"\t\"encoding/json\"\n" +
			"\t\"net/http\"\n" +
			")\n\n" +
			"type List" + Model + "sRequest struct{}\n\n" +
			"type Get" + Model + "Request struct {\n\tId string `json:\"id\"`\n}\n\n" +
			"type Delete" + Model + "Request struct {\n\tId string `json:\"id\"`\n}\n\n" +
			"//Remove the attribute that is not required for create or update as part of request\n" +
			"type Create" + Model + "Request struct {\n" +
			b.String() +
			"}\n\n" +
			"type Update" + Model + "Request struct {\n" +
			b.String() +
			"}\n\n" +
			make_decoder()

	return []byte(data)
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

func service_implementation_functions() string {
	data :=
		"func (s ServiceImpl) List" + Model + "s() ([]entity." + Model + ", error) {\n" +
			"\treturn s." + LowerCaseModel + "Repo.List" + Model + "s()\n" +
			"}\n\n" +

			"func (s ServiceImpl) Get" + Model + "(id string) (entity." + Model + ", error) {\n" +
			"\treturn s." + LowerCaseModel + "Repo.Get" + Model + "(id)\n" +
			"}\n\n" +

			"func (s ServiceImpl) Create" + Model + "(params domain.Create" + Model + "Params) (entity." + Model + ", error) {\n" +
			"\treturn s." + LowerCaseModel + "Repo.Create" + Model + "(" + LowerCaseModel + ")\n" +
			"}\n\n" +

			"func (s ServiceImpl) Update" + Model + "(params domain.Update" + Model + "Params) (entity." + Model + ", error) {\n" +
			"\treturn s." + LowerCaseModel + "Repo.Update" + Model + "(" + LowerCaseModel + ")\n" +
			"}\n\n" +

			"func (s ServiceImpl) Delete" + Model + "(id string) error {\n" +
			"\treturn s." + LowerCaseModel + "Repo.Delete" + Model + "(id)\n" +
			"}\n\n"
	return data
}

func make_decoder() string {
	return "func MakeDecoder(request interface{}) func (_ context.Context, r *http.Request) (interface{}, error) {\n" +
		"\treturn func (_ context.Context, r *http.Request) (interface{}, error) {\n" +
		"\t\tif err := json.NewDecoder(r.Body).Decode(&request); err != nil {\n" +
		"\t\t\treturn nil, err\n" +
		"\t\t}\n" +
		"\treturn request, nil\n" +
		"\t}\n" +
		"}"
}

func LowerInitial(str string) string {
	for i, v := range str {
		temp := string(unicode.ToLower(v)) + str[i+1:]
		return strings.Trim(temp, " ")
	}
	return ""
}
