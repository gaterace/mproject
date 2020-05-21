// Copyright 2019 Demian Harvill
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command line gRPC client for MServiceProject.
package main

import (
	"context"
	"encoding/json"

	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"regexp"
	"strconv"
	"strings"

	"github.com/gaterace/dml-go/pkg/dml"
	pb "github.com/gaterace/mproject/pkg/mserviceproject"
	"github.com/kylelemons/go-gypsy/yaml"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"

	flag "github.com/juju/gnuflag"
)

var dateValidator = regexp.MustCompile("^\\d\\d\\d\\d\\-\\d\\d\\-\\d\\d$")
var idlistValidator = regexp.MustCompile("^\\d+(,\\d+)*$")

var name = flag.String("name", "", "name")
var description = flag.String("desc", "", "description")
var sdate = flag.String("sdate", "", "start date")
var edate = flag.String("edate", "", "end date")
var email = flag.String("email", "", "email")
var hours = flag.String("hours", "", "hours")
var list = flag.String("list", "", "list  of ids")
var pid = flag.Int64("pid", -1, "project identifier")
var sid = flag.Int64("sid", -1, "status identifier")
var rid = flag.Int64("rid", -1, "role identifier")
var tid = flag.Int64("tid", -1, "task identifier")
var mid = flag.Int64("mid", -1, "member identifier")
var version = flag.Int64("version", -1, "version")
var priority = flag.Int64("priority", -1, "priority")
var parent = flag.Int64("parent", 0, "parent id")
var position = flag.Int64("position", -1, "position")

func main() {
	flag.Parse(true)

	configFilename := "conf.yaml"
	usr, err := user.Current()
	if err == nil {
		homeDir := usr.HomeDir
		configFilename = homeDir + string(os.PathSeparator) + ".mproject.config"
		// _ = homeDir + string(os.PathSeparator) + ".mproject.config"
	}

	config, err := yaml.ReadFile(configFilename)
	if err != nil {
		log.Fatalf("configuration not found: " + configFilename)
	}

	// log_file, _ := config.Get("log_file")
	ca_file, _ := config.Get("ca_file")
	tls, _ := config.GetBool("tls")
	server_host_override, _ := config.Get("server_host_override")
	server, _ := config.Get("server")
	port, _ := config.GetInt("port")

	if port == 0 {
		port = 50054
	}

	if len(flag.Args()) < 1 {
		prog := os.Args[0]
		fmt.Printf("Command line client for mproject grpc service\n")
		fmt.Printf("usage:\n")
		fmt.Printf("    %s create_project --name <name> --desc <description> --sid <status_id> --sdate <start_date> --edate <end_date>\n", prog)
		fmt.Printf("    %s update_project --pid <project_id> [--name <name>] [--desc <description>] [--sid <status_id>] [--sdate <start_date>] [--edate <end_date>]\n", prog)
		fmt.Printf("    %s delete_project --pid <project_id> --version <version>\n", prog)
		fmt.Printf("    %s get_project_names\n", prog)
		fmt.Printf("    %s get_project_by_name --name <name>\n", prog)
		fmt.Printf("    %s get_project_by_id --pid <project_id>\n", prog)
		fmt.Printf("    %s get_project_wrapper_by_id --pid <project_id>\n", prog)
		fmt.Printf("    %s get_project_wrapper_by_name --name <name>\n", prog)

		fmt.Printf("    %s create_status_type --sid <status_id>  --name <status_name> --desc <description>\n", prog)
		fmt.Printf("    %s update_status_type --sid <status_id> --version <version> --name <status_name> --desc <description>\n", prog)
		fmt.Printf("    %s delete_status_type --sid <status_id> --version <version>\n", prog)
		fmt.Printf("    %s get_status_type --sid <status_id> \n", prog)
		fmt.Printf("    %s get_status_types \n", prog)

		fmt.Printf("    %s create_project_role_type --rid <role_id>  --name <name> --desc <description>\n", prog)
		fmt.Printf("    %s update_project_role_type --rid <role_id>  --version <version>  --name <name> --desc <description>\n", prog)
		fmt.Printf("    %s delete_project_role_type --rid <role_id>  --version <version>\n", prog)
		fmt.Printf("    %s get_project_role_type --rid <role_id>\n", prog)
		fmt.Printf("    %s get_project_role_types\n", prog)

		fmt.Printf("    %s create_task --pid <project_id> --name <name> --desc <description> --sid <status_id> --sdate <start_date> --edate <end_date>\n", prog)
		fmt.Printf("        --priority <priority> [--parent <parent>] --position <position>\n")

		fmt.Printf("    %s update_task --tid <task_id> [--name <name>] [--desc <description>] [--sid <status_id>] [--sdate <start_date>] [--edate <end_date>]\n", prog)
		fmt.Printf("        [--priority <priority>] [--position <position>]\n")
		fmt.Printf("    %s delete_task --tid <task_id> --version <version>\n", prog)
		fmt.Printf("    %s get_task_by_id --tid <task_id>\n", prog)
		fmt.Printf("    %s get_tasks_by_project --pid <project_id>\n", prog)
		fmt.Printf("    %s get_task_wrapper_by_id --tid <task_id>\n", prog)
		fmt.Printf("    %s reorder_child_tasks --tid <task_id> --list <child_id_list>\n", prog)

		fmt.Printf("    %s create_team_member --pid <project_id> --name <name> --email <email> --rid <role_id>\n", prog)
		fmt.Printf("    %s update_team_member --mid <member_id> --version <version> --name <name> --email <email> --rid <role_id>\n", prog)
		fmt.Printf("    %s delete_team_member --mid <member_id> --version <version>\n", prog)
		fmt.Printf("    %s get_team_member_by_id --mid <member_id>\n", prog)
		fmt.Printf("    %s get_team_member_by_project --pid <project_id>\n", prog)
		fmt.Printf("    %s get_team_member_by_task --tid <task_id>\n", prog)
		fmt.Printf("    %s add_team_member_to_task --tid <task_id> --mid <member_id>\n", prog)
		fmt.Printf("    %s remove_team_member_from_task --tid <task_id> --mid <member_id>\n", prog)
		fmt.Printf("    %s add_task_hours --tid <task_id> --mid <member_id> --hours <hours>\n", prog)

		fmt.Printf("    %s get_server_version\n", prog)

		os.Exit(1)
	}

	cmd := flag.Arg(0)

	validParams := true

	// var id int64
	//  status_id int32
	// var role_id int32
	// var name string
	// var description string
	var start_date *dml.DateTime
	var end_date *dml.DateTime
	// var project_id int64
	var task_hours *dml.Decimal
	var id_list []int64

	switch cmd {
	case "create_project":

		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}

		if *description == "" {
			fmt.Println("description parameter missing")
			validParams = false
		}

		if *sid == -1 {
			fmt.Println("status_id parameter missing")
			validParams = false
		}

		date := *sdate
		if !dateValidator.MatchString(date) {
			fmt.Println("start_date parameter missing or not in yyyy-mm-dd format")
			validParams = false
		}

		start_date = dml.DateTimeFromString(date)

		date = *edate
		if !dateValidator.MatchString(date) {
			fmt.Println("end_date parameter missing or not in yyyy-mm-dd format")
			validParams = false
		}

		end_date = dml.DateTimeFromString(date)

	case "update_project":
		if *pid == -1 {
			fmt.Println("project_id parameter missing")
			validParams = false
		}

		date := *sdate
		if date == "" {
			start_date = nil
		} else {
			if !dateValidator.MatchString(date) {
				fmt.Println("start_date parameter missing or not in yyyy-mm-dd format")
				validParams = false
			}

			start_date = dml.DateTimeFromString(date)
		}

		date = *edate
		if date == "" {
			end_date = nil
		} else {
			if !dateValidator.MatchString(date) {
				fmt.Println("start_date parameter missing or not in yyyy-mm-dd format")
				validParams = false
			}

			end_date = dml.DateTimeFromString(date)
		}

	case "delete_project":
		if *pid == -1 {
			fmt.Println("project_id parameter missing")
			validParams = false
		}

		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
	case "get_project_names":
		// no params

	case "get_project_by_name":
		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}

	case "get_project_by_id":
		if *pid == -1 {
			fmt.Println("project_id parameter missing")
			validParams = false
		}

	case "get_project_wrapper_by_id":
		if *pid == -1 {
			fmt.Println("project_id parameter missing")
			validParams = false
		}

	case "get_project_wrapper_by_name":
		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}
	case "create_status_type":

		if *sid == -1 {
			fmt.Println("status_id parameter missing")
			validParams = false
		}

		if *name == "" {
			fmt.Println("status_name parameter missing")
			validParams = false
		}

		if *description == "" {
			fmt.Println("description parameter missing")
			validParams = false
		}

	case "update_status_type":
		if *sid == -1 {
			fmt.Println("status_id parameter missing")
			validParams = false
		}

		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

		if *name == "" {
			fmt.Println("status_name parameter missing")
			validParams = false
		}

		if *description == "" {
			fmt.Println("description parameter missing")
			validParams = false
		}

	case "delete_status_type":
		if *sid == -1 {
			fmt.Println("status_id parameter missing")
			validParams = false
		}

		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

	case "get_status_type":
		if *sid == -1 {
			fmt.Println("status_id parameter missing")
			validParams = false
		}

	case "get_status_types":
		// OK

	case "create_project_role_type":
		if *rid == -1 {
			fmt.Println("role_id parameter missing")
			validParams = false
		}

		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}

		if *description == "" {
			fmt.Println("description parameter missing")
			validParams = false
		}

	case "update_project_role_type":
		if *rid == -1 {
			fmt.Println("role_id parameter missing")
			validParams = false
		}

		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}

		if *description == "" {
			fmt.Println("description parameter missing")
			validParams = false
		}

	case "delete_project_role_type":
		if *rid == -1 {
			fmt.Println("role_id parameter missing")
			validParams = false
		}

		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

	case "get_project_role_type":
		if *rid == -1 {
			fmt.Println("role_id parameter missing")
			validParams = false
		}

	case "get_project_role_types":
		// OK

	case "create_task":
		if *pid == -1 {
			fmt.Println("project_id parameter missing")
			validParams = false
		}
		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}

		if *description == "" {
			fmt.Println("description parameter missing")
			validParams = false
		}

		if *sid == -1 {
			fmt.Println("status_id parameter missing")
			validParams = false
		}

		date := *sdate
		if !dateValidator.MatchString(date) {
			fmt.Println("start_date parameter missing or not in yyyy-mm-dd format")
			validParams = false
		}

		start_date = dml.DateTimeFromString(date)

		date = *edate
		if !dateValidator.MatchString(date) {
			fmt.Println("end_date parameter missing or not in yyyy-mm-dd format")
			validParams = false
		}

		end_date = dml.DateTimeFromString(date)

		if (*priority < 1) || (*priority > 9) {
			fmt.Println("priority parameter missing or not in range [1,9] ")
			validParams = false
		}

		if *position == -1 {
			fmt.Println("position parameter missing")
			validParams = false
		}

	case "update_task":
		if *tid == -1 {
			fmt.Println("task_id parameter missing")
			validParams = false
		}
		date := *sdate
		if date == "" {
			start_date = nil
		} else {
			if !dateValidator.MatchString(date) {
				fmt.Println("start_date parameter missing or not in yyyy-mm-dd format")
				validParams = false
			}

			start_date = dml.DateTimeFromString(date)
		}

		date = *edate
		if date == "" {
			end_date = nil
		} else {
			if !dateValidator.MatchString(date) {
				fmt.Println("start_date parameter missing or not in yyyy-mm-dd format")
				validParams = false
			}

			end_date = dml.DateTimeFromString(date)
		}

		if *priority != -1 {
			if (*priority < 1) || (*priority > 9) {
				fmt.Println("priority parameter missing or not in range [1,9] ")
				validParams = false
			}
		}

	case "delete_task":
		if *tid == -1 {
			fmt.Println("task_id parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

	case "get_task_by_id":
		if *tid == -1 {
			fmt.Println("task_id parameter missing")
			validParams = false
		}

	case "get_tasks_by_project":
		if *pid == -1 {
			fmt.Println("project_id parameter missing")
			validParams = false
		}

	case "get_task_wrapper_by_id":
		if *tid == -1 {
			fmt.Println("task_id parameter missing")
			validParams = false
		}

	case "reorder_child_tasks":
		if *tid == -1 {
			fmt.Println("task_id parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version  parameter missing")
			validParams = false
		}

		if idlistValidator.MatchString(*list) {
			ids := strings.Split(*list, ",")
			for _, s := range ids {
				id, err := strconv.ParseInt(s, 10, 64)
				if err == nil {
					id_list = append(id_list, id)
				}
			}
		} else {
			fmt.Println("list parameter missing or not comma sepated list of ids")
			validParams = false
		}

	case "create_team_member":
		if *pid == -1 {
			fmt.Println("project_id parameter missing")
			validParams = false
		}

		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}

		if *email == "" {
			fmt.Println("email parameter missing")
			validParams = false
		}
		if *rid == -1 {
			fmt.Println("role_id parameter missing")
			validParams = false
		}

	case "update_team_member":
		if *mid == -1 {
			fmt.Println("member_id parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}
		if *name == "" {
			fmt.Println("name parameter missing")
			validParams = false
		}

		if *email == "" {
			fmt.Println("email parameter missing")
			validParams = false
		}
		if *rid == -1 {
			fmt.Println("role_id parameter missing")
			validParams = false
		}

	case "delete_team_member":
		if *mid == -1 {
			fmt.Println("member_id parameter missing")
			validParams = false
		}
		if *version == -1 {
			fmt.Println("version parameter missing")
			validParams = false
		}

	case "get_team_member_by_id":
		if *mid == -1 {
			fmt.Println("member_id parameter missing")
			validParams = false
		}

	case "get_team_member_by_project":
		if *pid == -1 {
			fmt.Println("project_id parameter missing")
			validParams = false
		}

	case "get_team_member_by_task":
		if *tid == -1 {
			fmt.Println("task_id parameter missing")
			validParams = false
		}
	case "add_team_member_to_task":
		if *mid == -1 {
			fmt.Println("member_id parameter missing")
			validParams = false
		}
		if *tid == -1 {
			fmt.Println("task_id parameter missing")
			validParams = false
		}

	case "remove_team_member_from_task":
		if *mid == -1 {
			fmt.Println("member_id parameter missing")
			validParams = false
		}
		if *tid == -1 {
			fmt.Println("task_id parameter missing")
			validParams = false
		}
	case "add_task_hours":
		if *mid == -1 {
			fmt.Println("member_id parameter missing")
			validParams = false
		}
		if *tid == -1 {
			fmt.Println("task_id parameter missing")
			validParams = false
		}
		if *hours == "" {
			fmt.Println("hours parameter missing")
			validParams = false
		} else {
			task_hours, err = dml.DecimalFromString(*hours)
			if err != nil {
				fmt.Println("hours parameter not valid")
				validParams = false
			}
		}
	case "get_server_version":
		validParams = true

	default:
		fmt.Printf("unknown command: %s\n", cmd)
		validParams = false
	}

	if !validParams {
		os.Exit(1)
	}

	tokenFilename := "token.txt"
	usr, err = user.Current()
	if err == nil {
		homeDir := usr.HomeDir
		tokenFilename = homeDir + string(os.PathSeparator) + ".mservice.token"
	}

	address := server + ":" + strconv.Itoa(int(port))
	// fmt.Printf("address: %s\n", address)

	var opts []grpc.DialOption
	if tls {
		var sn string
		if server_host_override != "" {
			sn = server_host_override
		}
		var creds credentials.TransportCredentials
		if ca_file != "" {
			var err error
			creds, err = credentials.NewClientTLSFromFile(ca_file, sn)
			if err != nil {
				grpclog.Fatalf("Failed to create TLS credentials %v", err)
			}
		} else {
			creds = credentials.NewClientTLSFromCert(nil, sn)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	// set up connection to server
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer conn.Close()

	client := pb.NewMServiceProjectClient(conn)

	ctx := context.Background()

	savedToken := ""

	data, err := ioutil.ReadFile(tokenFilename)

	if err == nil {
		savedToken = string(data)
	}

	md := metadata.Pairs("token", savedToken)
	mctx := metadata.NewOutgoingContext(ctx, md)

	switch cmd {
	case "create_project":
		req := pb.CreateProjectRequest{}
		req.Name = *name
		req.Description = *description
		req.StatusId = int32(*sid)
		req.StartDate = start_date
		req.EndDate = end_date
		resp, err := client.CreateProject(mctx, &req)
		printResponse(resp, err)

	case "update_project":
		req1 := pb.GetProjectByIdRequest{}
		req1.ProjectId = *pid
		resp1, err := client.GetProjectById(mctx, &req1)
		if err == nil {
			if resp1.GetErrorCode() == 0 {
				req2 := pb.UpdateProjectRequest{}
				req2.ProjectId = *pid
				req2.Version = resp1.GetProject().GetVersion()
				if *name == "" {
					req2.Name = resp1.GetProject().GetName()
				} else {
					req2.Name = *name
				}
				if *description == "" {
					req2.Description = resp1.GetProject().GetDescription()
				} else {
					req2.Description = *description
				}

				if *sid == -1 {
					req2.StatusId = resp1.GetProject().GetStatusId()
				} else {
					req2.StatusId = int32(*sid)
				}
				if start_date == nil {
					req2.StartDate = resp1.GetProject().GetStartDate()
				} else {
					req2.StartDate = start_date
				}
				if end_date == nil {
					req2.EndDate = resp1.GetProject().GetEndDate()
				} else {
					req2.EndDate = end_date
				}

				resp2, err := client.UpdateProject(mctx, &req2)
				if err == nil {
					jtext, err := json.MarshalIndent(resp2, "", "  ")
					if err == nil {
						fmt.Println(string(jtext))
					}
				}
				if err != nil {
					fmt.Printf("err: %s\n", err)
				}

			} else {
				jtext, err := json.MarshalIndent(resp1, "", "  ")
				if err == nil {
					fmt.Println(string(jtext))
				}
			}
		} else {
			fmt.Printf("err: %s\n", err)
		}

	case "delete_project":
		req := pb.DeleteProjectRequest{}
		req.ProjectId = *pid
		req.Version = int32(*version)
		resp, err := client.DeleteProject(mctx, &req)
		printResponse(resp, err)

	case "get_project_names":
		req := pb.GetProjectNamesRequest{}
		resp, err := client.GetProjectNames(mctx, &req)
		printResponse(resp, err)

	case "get_project_by_name":
		req := pb.GetProjectByNameRequest{}
		req.Name = *name
		resp, err := client.GetProjectByName(mctx, &req)
		printResponse(resp, err)

	case "get_project_by_id":
		req := pb.GetProjectByIdRequest{}
		req.ProjectId = *pid
		resp, err := client.GetProjectById(mctx, &req)
		printResponse(resp, err)

	case "get_project_wrapper_by_id":
		req := pb.GetProjectWrapperByIdRequest{}
		req.ProjectId = *pid
		resp, err := client.GetProjectWrapperById(mctx, &req)
		printResponse(resp, err)

	case "get_project_wrapper_by_name":
		req := pb.GetProjectWrapperByNameRequest{}
		req.Name = *name
		resp, err := client.GetProjectWrapperByName(mctx, &req)
		printResponse(resp, err)

	case "create_status_type":
		req := pb.CreateStatusTypeRequest{}
		req.StatusId = int32(*sid)
		req.StatusName = *name
		req.Description = *description
		resp, err := client.CreateStatusType(mctx, &req)
		printResponse(resp, err)

	case "update_status_type":
		req := pb.UpdateStatusTypeRequest{}
		req.StatusId = int32(*sid)
		req.Version = int32(*version)
		req.StatusName = *name
		req.Description = *description
		resp, err := client.UpdateStatusType(mctx, &req)
		printResponse(resp, err)

	case "delete_status_type":
		req := pb.DeleteStatusTypeRequest{}
		req.StatusId = int32(*sid)
		req.Version = int32(*version)

		resp, err := client.DeleteStatusType(mctx, &req)
		printResponse(resp, err)

	case "get_status_type":
		req := pb.GetStatusTypeRequest{}
		req.StatusId = int32(*sid)

		resp, err := client.GetStatusType(mctx, &req)
		printResponse(resp, err)

	case "get_status_types":
		req := pb.GetStatusTypesRequest{}
		resp, err := client.GetStatusTypes(mctx, &req)
		printResponse(resp, err)

	case "create_project_role_type":
		req := pb.CreateProjectRoleTypeRequest{}
		req.ProjectRoleId = int32(*rid)
		req.RoleName = *name
		req.Description = *description
		resp, err := client.CreateProjectRoleType(mctx, &req)
		printResponse(resp, err)

	case "get_project_role_type":
		req := pb.GetProjectRoleTypeRequest{}
		req.ProjectRoleId = int32(*rid)
		resp, err := client.GetProjectRoleType(mctx, &req)
		printResponse(resp, err)

	case "update_project_role_type":
		req := pb.UpdateProjectRoleTypeRequest{}
		req.ProjectRoleId = int32(*rid)
		req.Version = int32(*version)
		req.RoleName = *name
		req.Description = *description
		resp, err := client.UpdateProjectRoleType(mctx, &req)
		printResponse(resp, err)

	case "delete_project_role_type":
		req := pb.DeleteProjectRoleTypeRequest{}
		req.ProjectRoleId = int32(*rid)
		req.Version = int32(*version)
		resp, err := client.DeleteProjectRoleType(mctx, &req)
		printResponse(resp, err)

	case "get_project_role_types":
		req := pb.GetProjectRoleTypesRequest{}
		resp, err := client.GetProjectRoleTypes(mctx, &req)
		printResponse(resp, err)

	case "create_task":
		req := pb.CreateTaskRequest{}
		req.ProjectId = *pid
		req.Name = *name
		req.Description = *description
		req.StatusId = int32(*sid)
		req.StartDate = start_date
		req.EndDate = end_date
		req.Priority = int32(*priority)
		req.ParentId = *parent
		req.Position = int32(*position)
		resp, err := client.CreateTask(mctx, &req)
		printResponse(resp, err)

	case "update_task":
		var task *pb.Task

		// get the current version
		req1 := pb.GetTaskByIdRequest{}
		req1.TaskId = *tid
		resp1, err := client.GetTaskById(mctx, &req1)
		if err == nil {
			if resp1.GetErrorCode() == 0 {
				task = resp1.GetTask()
			} else {
				jtext, err := json.MarshalIndent(resp1, "", "  ")
				if err == nil {
					fmt.Println(string(jtext))
				}
			}

		}

		if task != nil {
			req2 := pb.UpdateTaskRequest{}
			req2.TaskId = *tid
			req2.Version = task.GetVersion()
			if *name == "" {
				req2.Name = task.GetName()
			} else {
				req2.Name = *name
			}

			if *description == "" {
				req2.Description = task.GetDescription()
			} else {
				req2.Description = *description
			}

			if *sid == -1 {
				req2.StatusId = task.GetStatusId()
			} else {
				req2.StatusId = int32(*sid)
			}

			if start_date == nil {
				req2.StartDate = task.GetStartDate()
			} else {
				req2.StartDate = start_date
			}

			if end_date == nil {
				req2.EndDate = task.GetEndDate()
			} else {
				req2.EndDate = end_date
			}

			if *priority == -1 {
				req2.Priority = task.GetPriority()
			} else {
				req2.Priority = int32(*priority)
			}

			req2.ParentId = task.GetParentId()

			if *position == -1 {
				req2.Position = task.GetPosition()
			} else {
				req2.Position = int32(*position)
			}

			resp2, err := client.UpdateTask(mctx, &req2)
			if err == nil {
				jtext, err := json.MarshalIndent(resp2, "", "  ")
				if err == nil {
					fmt.Println(string(jtext))
				}
			}

		}

		if err != nil {
			fmt.Printf("err: %s\n", err)
		}

	case "delete_task":
		req := pb.DeleteTaskRequest{}
		req.TaskId = *tid
		req.Version = int32(*version)

		resp, err := client.DeleteTask(mctx, &req)
		printResponse(resp, err)

	case "get_task_by_id":
		req := pb.GetTaskByIdRequest{}
		req.TaskId = *tid
		resp, err := client.GetTaskById(mctx, &req)
		printResponse(resp, err)

	case "get_tasks_by_project":
		req := pb.GetTasksByProjectRequest{}
		req.ProjectId = *pid
		resp, err := client.GetTasksByProject(mctx, &req)
		printResponse(resp, err)

	case "get_task_wrapper_by_id":
		req := pb.GetTaskWrapperByIdRequest{}
		req.TaskId = *tid
		resp, err := client.GetTaskWrapperById(mctx, &req)
		printResponse(resp, err)

	case "reorder_child_tasks":
		req := pb.ReorderChildTasksRequest{}
		req.TaskId = *tid
		req.Version = int32(*version)
		req.ChildTaskIds = id_list
		resp, err := client.ReorderChildTasks(mctx, &req)
		printResponse(resp, err)

	case "create_team_member":
		req := pb.CreateTeamMemberRequest{}
		req.ProjectId = *pid
		req.Name = *name
		req.Email = *email
		req.ProjectRoleId = int32(*rid)
		resp, err := client.CreateTeamMember(mctx, &req)
		printResponse(resp, err)

	case "update_team_member":
		req := pb.UpdateTeamMemberRequest{}
		req.MemberId = *mid
		req.Version = int32(*version)
		req.Name = *name
		req.Email = *email
		req.ProjectRoleId = int32(*rid)
		resp, err := client.UpdateTeamMember(mctx, &req)
		printResponse(resp, err)

	case "delete_team_member":
		req := pb.DeleteTeamMemberRequest{}
		req.MemberId = *mid
		req.Version = int32(*version)
		resp, err := client.DeleteTeamMember(mctx, &req)
		printResponse(resp, err)

	case "get_team_member_by_id":
		req := pb.GetTeamMemberByIdRequest{}
		req.MemberId = *mid
		resp, err := client.GetTeamMemberById(mctx, &req)
		printResponse(resp, err)

	case "get_team_member_by_project":
		req := pb.GetTeamMemberByProjectRequest{}
		req.ProjectId = *pid
		resp, err := client.GetTeamMemberByProject(mctx, &req)
		printResponse(resp, err)

	case "get_team_member_by_task":
		req := pb.GetTeamMemberByTaskRequest{}
		req.TaskId = *tid
		resp, err := client.GetTeamMemberByTask(mctx, &req)
		printResponse(resp, err)

	case "add_team_member_to_task":
		req := pb.AddTeamMemberToTaskRequest{}
		req.MemberId = *mid
		req.TaskId = *tid
		resp, err := client.AddTeamMemberToTask(mctx, &req)
		printResponse(resp, err)

	case "remove_team_member_from_task":
		req := pb.RemoveTeamMemberFromTaskRequest{}
		req.MemberId = *mid
		req.TaskId = *tid
		resp, err := client.RemoveTeamMemberFromTask(mctx, &req)
		printResponse(resp, err)

	case "add_task_hours":
		req := pb.AddTaskHoursRequest{}
		req.MemberId = *mid
		req.TaskId = *tid
		req.TaskHours = task_hours
		resp, err := client.AddTaskHours(mctx, &req)
		printResponse(resp, err)
	case "get_server_version":
		req := pb.GetServerVersionRequest{}
		req.DummyParam = 1
		resp, err := client.GetServerVersion(mctx, &req)
		printResponse(resp, err)
	default:
		fmt.Printf("unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

// Helper to print api method response as JSON.
func printResponse(resp interface{}, err error) {
	if err == nil {
		jtext, err := json.MarshalIndent(resp, "", "  ")
		if err == nil {
			fmt.Println(string(jtext))
		}
	}
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
}
