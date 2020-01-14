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
	readFile(filename)
	LowerCaseModel = LowerInitial(Model)
	r := strings.Split(Service, "-")
	for _, value := range r {
		ServiceAbbreviation += string(value[0])
	}
	fmt.Println(ServiceAbbreviation)

	// Domain Structure
	domainDir := Service + "/" + "domain/entity"
	createDir(domainDir)
	createFile(domainDir, ToSnakeCase(Model)+".go", entityModelData())
	createFile(Service+"/"+"domain", "service.go", serviceInterfaceData())

	// Service structure
	serviceDir := Service + "/service"
	createDir(serviceDir)
	createFile(serviceDir, "service.go", serviceImplementation())

	// Endpoint structure
	endpointDir := Service + "/endpoint"
	createDir(endpointDir)
	createFile(endpointDir, "decoder.go", decoderData())
	createFile(endpointDir, "encoder.go", encoderData())
	createFile(endpointDir, "view.go", viewData())
	createFile(endpointDir, "endpoint.go", endpointData())

	// Repository structure
	repositoryDir := Service + "/repository/impl/postgresql"
	createDir(repositoryDir)
	createFile(Service+"/repository", ToSnakeCase(Model)+".go", repoInterfaceData())
	createFile(repositoryDir, "connection.go", connectionData())
	createFile(repositoryDir, ToSnakeCase(Model)+".go", repoImplementationData())

	//main.go
	createFile(Service, "main.go", mainData())
}

func createDir(dirName string) error {
	err := os.MkdirAll(dirName, 0777)
	if err == nil || os.IsExist(err) {
		return nil
	} else {
		log.Fatal(err)
		return err
	}
}

func createFile(dirPath string, name string, data []byte) {
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

func readFile(name string) {
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

func entityModelData() []byte {
	b := new(bytes.Buffer)
	for key, value := range Attributes {
		fmt.Fprintf(b, "\t%s %s\n", key, value)
	}

	data := "package entity\n\ntype " + Model + " struct {\n" + b.String() + "}"
	return []byte(data)
}

func serviceInterfaceData() []byte {
	b := new(bytes.Buffer)
	for key, value := range Attributes {
		fmt.Fprintf(b, "\t%s %s\n", key, value)
	}

	data :=
		"package domain\n\n" +
			"import \"" + Service + "/domain/entity\"\n\n" +
			"type Service interface {\n" +
			serviceFunctions("entity.", "") +
			"}\n\n" +
			"//Please remove the attribute that is not required for create or update params\n\n" +
			"type Create" + Model + "Params struct {\n" + b.String() + "}\n\n" +
			"type Update" + Model + "Params struct {\n" + b.String() + "}"

	return []byte(data)
}

func repoInterfaceData() []byte {
	b := new(bytes.Buffer)
	for key, value := range Attributes {
		fmt.Fprintf(b, "\t%s %s\n", key, value)
	}

	data :=
		"package repository\n\n" +
			"import \"" + Service + "/domain\"\n\n" +
			"import \"" + Service + "/domain/entity\"\n\n" +
			"type " + Model + "Repo interface {\n" +
			serviceFunctions("entity.", "domain.") +
			"}\n\n"

	return []byte(data)
}

// func repos_models_data() []byte {
// 	data := "package repos\n\n" +
// 		"import " + ServiceAbbreviation + " \"" + Service + "/service\"\n\n" +
// 		"type " + Model + "Repo " + "interface {\n" +
// 		serviceFunctions(ServiceAbbreviation+".") + "}"

// 	return []byte(data)
// }

func serviceImplementation() []byte {
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
			serviceImplementationFunctions()

	return []byte(data)
}

func serviceFunctions(prefix string, prefix_2 string) string {
	list := "\tList" + Model + "s() ([]" + prefix + Model + ", error)\n"
	get := "\tGet" + Model + "(id string) (" + prefix + Model + ", error)\n"
	create := "\tCreate" + Model + "(params " + prefix_2 + "Create" + Model + "Params) (" + prefix + Model + ", error)\n"
	update := "\tUpdate" + Model + "(params " + prefix_2 + "Update" + Model + "Params) (" + prefix + Model + ", error)\n"
	del := "\tDelete" + Model + "(id string) (interface{}, error)\n"

	return list + get + create + update + del
}

func decoderData() []byte {
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
			makeDecoder()

	return []byte(data)
}

func encoderData() []byte {
	data := "package endpoint\n\n" +
		"import (\n" +
		"\t\"context\"\n" +
		"\t\"encoding/json\"\n" +
		"\t\"net/http\"\n" +
		")\n\n" +
		"type Response struct {\n" +
		"\tData interface{} `json:\"data\"`\n" +
		"\tErrors []error `json:\"errors\"`\n" +
		"}\n\n" +
		"func EncodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {\n" +
		"\treturn json.NewEncoder(w).Encode(response)\n" +
		"}"

	return []byte(data)
}

func viewData() []byte {
	b := new(bytes.Buffer)
	v := new(bytes.Buffer)
	for key, value := range Attributes {
		fmt.Fprintf(b, "\t%s %s %s\n", key, value, "`json:\""+LowerInitial(key)+"\"`")
		fmt.Fprintf(v, "\t\t%s: %s\n", key, LowerCaseModel+"."+key+",")
	}

	data := "package endpoint\n\n" +
		"import (\n" +
		"\t\"" + Service + "/domain/entity\"\n" +
		")\n\n" +
		"type " + Model + "View struct {\n" +
		b.String() +
		"}\n\n" +

		"func to" + Model + "View(" + LowerCaseModel + "entity." + Model + ") " + Model + "View {\n" +
		"\treturn " + Model + "View{\n" +
		v.String() +
		"\t}\n}"

	return []byte(data)
}

func endpointData() []byte {
	data := "package endpoint\n\n" +
		"import (\n" +
		"\t\"" + Service + "/domain\"\n" +
		"\t\"context\"\n" +
		"\t\"github.com/go-kit/kit/endpoint\"\n" +
		")\n\n" +
		endpointImplementationFunctions()

	return []byte(data)
}

func serviceImplementationFunctions() string {
	data :=
		"func (s ServiceImpl) List" + Model + "s() ([]entity." + Model + ", error) {\n" +
			"\treturn s." + LowerCaseModel + "Repo.List" + Model + "s()\n" +
			"}\n\n" +

			"func (s ServiceImpl) Get" + Model + "(id string) (entity." + Model + ", error) {\n" +
			"\treturn s." + LowerCaseModel + "Repo.Get" + Model + "(id)\n" +
			"}\n\n" +

			"func (s ServiceImpl) Create" + Model + "(params domain.Create" + Model + "Params) (entity." + Model + ", error) {\n" +
			"\treturn s." + LowerCaseModel + "Repo.Create" + Model + "(params)\n" +
			"}\n\n" +

			"func (s ServiceImpl) Update" + Model + "(params domain.Update" + Model + "Params) (entity." + Model + ", error) {\n" +
			"\treturn s." + LowerCaseModel + "Repo.Update" + Model + "(params)\n" +
			"}\n\n" +

			"func (s ServiceImpl) Delete" + Model + "(id string) error {\n" +
			"\treturn s." + LowerCaseModel + "Repo.Delete" + Model + "(id)\n" +
			"}\n\n"
	return data
}

func endpointImplementationFunctions() string {
	data :=
		funcString("MakeList"+Model+"sEndpoint") +
			"\t" + returnString() +
			"\t\t" + "v, err := s.List" + Model + "s()\n" +
			"\t\t" + errorNotNilString() +
			LowerCaseModel + "s" + " := make([]" + Model + "View, 0)\n" +
			"for _, " + LowerCaseModel + " := range v {\n" +
			"\t" + LowerCaseModel + "s" + " = append(" + LowerCaseModel + "s" + ", to" + Model + "View(" + LowerCaseModel + "))\n" +
			"}\n" +
			responseString(LowerCaseModel+"s", "nil") +
			"}\n}\n\n" +

			funcString("MakeGet"+Model+"Endpoint") +
			"\t" + returnString() +
			"\treq := request.(Get" + Model + "Request)\n" +
			"\t" + ServiceAbbreviation + ", err := s.Get" + Model + "(req.Id)\n" +
			"\t\t" + errorNotNilString() +
			responseString("to"+Model+"View("+ServiceAbbreviation+")", "nil") +
			"}\n}\n\n" +

			create_update("Create") +
			create_update("Update") +

			funcString("MakeDelete"+Model+"Endpoint") +
			"\t" + returnString() +
			"\treq := request.(Delete" + Model + "Request)\n" +
			"\terr := s.Delete" + Model + "(req.Id)\n" +
			"\t\t" + errorNotNilString() +
			responseString("nil", "nil") +
			"}\n}\n\n"

	return data
}

func create_update(name string) string {
	data :=
		funcString("Make"+name+Model+"Endpoint") +
			"\t" + returnString() +
			"\treq := request.(" + name + Model + "Request)\n" +
			"\t" + ServiceAbbreviation + ", err := s." + name + Model + "(domain." + name + Model + "Params(req))\n" +
			"\t\t" + errorNotNilString() +
			responseString("to"+Model+"View("+ServiceAbbreviation+")", "nil") +
			"}\n}\n\n"
	return data
}

func makeDecoder() string {
	return "func MakeDecoder(request interface{}) func (_ context.Context, r *http.Request) (interface{}, error) {\n" +
		"\treturn func (_ context.Context, r *http.Request) (interface{}, error) {\n" +
		"\t\tif err := json.NewDecoder(r.Body).Decode(&request); err != nil {\n" +
		"\t\t\treturn nil, err\n" +
		"\t\t}\n" +
		"\treturn request, nil\n" +
		"\t}\n" +
		"}"
}

func repoImplementationData() []byte {
	return []byte("Yet To BE ImpleMEnted")
}

func connectionData() []byte {
	data := `
package postgresql

import (
	"upper.io/db.v3/lib/sqlbuilder"
	"upper.io/db.v3/postgresql"
)

func openConn() sqlbuilder.Database {
	connSettings := postgresql.ConnectionURL{
		User:     "",
		Password: "",
		Host:     "",
		Socket:   "",
		Database: "",
		Options:  nil,
	}

	conn, err := postgresql.Open(connSettings)

	if err != nil {
		panic("SHIT NO DB")
	}

	return conn
}

func getReadConn() sqlbuilder.Database {
	return openConn();
}

func getWriteConn() sqlbuilder.Database {
	return openConn();
}
	`
	return []byte(data)
}

func mainData() []byte {
	data := "package main\n\n" +
		"import (\n" +
		"\t\"" + Service + "/service\"\n" +
		"\t\"" + Service + "/endpoint\"\n" +
		"\thttpTransport \"github.com/go-kit/kit/transport/http\"\n" +
		"\t\"github.com/gorilla/mux\"\n" +
		"\t\"log\"\n" +
		"\t\"net/http\"\n" +
		")\n\n"

	main := `func main() {
	router := mux.NewRouter()
	assignRoutes(router)
	http.Handle("/", router)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
`
	remain :=
		"\nfunc assignRoutes(router *mux.Router) *mux.Router {\n" +
			"\tservice := service.MakeServiceImpl()\n" +

			"\n\tlist" + Model + "sHandler := httpTransport.NewServer(\n" +
			"\tendpoint.MakeList" + Model + "sEndpoint(service),\n" +
			"\tendpoint.MakeDecoder(endpoint.List" + Model + "sRequest{}),\n" +
			"\tendpoint.EncodeResponse)\n" +

			"\n\trouter.Handle(\"/" + LowerCaseModel + "s\", list" + Model + "sHandler).Methods(\"GET\")\n" +
			"\treturn router\n}\n"

	return []byte(data + main + remain)
}

func LowerInitial(str string) string {
	for i, v := range str {
		temp := string(unicode.ToLower(v)) + str[i+1:]
		return strings.Trim(temp, " ")
	}
	return ""
}

func responseString(data string, err string) string {
	return "return Response{Data: " + data + ", Errors: " + err + "}, " + err + "\n"
}

func returnString() string {
	return "return func(_ context.Context, request interface{}) (interface{}, error) {\n"
}

func errorNotNilString() string {
	data := "if err != nil {\n" +
		"\treturn Response{Data: nil, Errors: []error{err}}, err\n" +
		"}\n"
	return data
}

func funcString(name string) string {
	return "func " + name + "(s domain.Service) endpoint.Endpoint {\n"
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
