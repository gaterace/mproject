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

// Package projauth provides authorization for each GRPC method in MServiceProject.
// The JWT extracted from the GRPC request context is used for each delegating method.
package projauth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"

	pb "github.com/gaterace/mproject/pkg/mserviceproject"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"crypto/rsa"
	"io/ioutil"
)

var NotImplemented = errors.New("not implemented")

type projAuth struct {
	logger          *log.Logger
	db              *sql.DB
	rsaPSSPublicKey *rsa.PublicKey
	projService     pb.MServiceProjectServer
}

// Get a new projAuth instance.
func NewProjectAuth(projService pb.MServiceProjectServer) *projAuth {
	svc := projAuth{}
	svc.projService = projService
	return &svc
}

// Set the logger for the projAuth instance.
func (s *projAuth) SetLogger(logger *log.Logger) {
	s.logger = logger
}

// Set the database connection for the projAuth instance.
func (s *projAuth) SetDatabaseConnection(sqlDB *sql.DB) {
	s.db = sqlDB
}

// Set the public RSA key for the projAuth instance, used to validate JWT.
func (s *projAuth) SetPublicKey(publicKeyFile string) error {
	publicKey, err := ioutil.ReadFile(publicKeyFile)
	if err != nil {
		s.logger.Printf("error reading publicKeyFile: %v\n", err)
		return err
	}

	parsedKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKey)
	if err != nil {
		s.logger.Printf("error parsing publicKeyFile: %v\n", err)
		return err
	}

	s.rsaPSSPublicKey = parsedKey
	return nil
}

// Bind our projAuth as the gRPC api server.
func (s *projAuth) NewApiServer(gServer *grpc.Server) error {
	if s != nil {
		pb.RegisterMServiceProjectServer(gServer, s)

	}
	return nil
}

// Get the JWT from the gRPC request context.
func (s *projAuth) GetJwtFromContext(ctx context.Context) (*map[string]interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("cannot get metadata from context")
	}

	tokens := md["token"]

	if (tokens == nil) || (len(tokens) == 0) {
		return nil, fmt.Errorf("cannot get token from context")
	}

	tokenString := tokens[0]

	s.logger.Printf("tokenString: %s\n", tokenString)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		method := token.Method.Alg()
		if method != "PS256" {

			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		// return []byte(mySigningKey), nil
		return s.rsaPSSPublicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid json web token")
	}

	claims := map[string]interface{}(token.Claims.(jwt.MapClaims))

	s.logger.Printf("claims: %v\n", claims)

	return &claims, nil

}

// Get the clain value as an int64.
func GetInt64FromClaims(claims *map[string]interface{}, key string) int64 {
	var val int64

	if claims != nil {
		cval := (*claims)[key]
		if fval, ok := cval.(float64); ok {
			val = int64(fval)
		}
	}

	return val
}

// Get the claim value as a string.
func GetStringFromClaims(claims *map[string]interface{}, key string) string {
	var val string

	if claims != nil {
		cval := (*claims)[key]
		if sval, ok := cval.(string); ok {
			val = sval
		}
	}

	return val
}

// create a new project
func (s *projAuth) CreateProject(ctx context.Context, req *pb.CreateProjectRequest) (*pb.CreateProjectResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.CreateProject(ctx, req)
		}
	}
	resp := &pb.CreateProjectResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// update an existing project
func (s *projAuth) UpdateProject(ctx context.Context, req *pb.UpdateProjectRequest) (*pb.UpdateProjectResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.UpdateProject(ctx, req)
		}
	}
	resp := &pb.UpdateProjectResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// delete an existing project
func (s *projAuth) DeleteProject(ctx context.Context, req *pb.DeleteProjectRequest) (*pb.DeleteProjectResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.DeleteProject(ctx, req)
		}
	}
	resp := &pb.DeleteProjectResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get list of project names for this mservice id
func (s *projAuth) GetProjectNames(ctx context.Context, req *pb.GetProjectNamesRequest) (*pb.GetProjectNamesResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.GetProjectNames(ctx, req)
		}
	}
	resp := &pb.GetProjectNamesResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get project entity by name
func (s *projAuth) GetProjectByName(ctx context.Context, req *pb.GetProjectByNameRequest) (*pb.GetProjectByNameResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.GetProjectByName(ctx, req)
		}
	}
	resp := &pb.GetProjectByNameResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get project entity by id
func (s *projAuth) GetProjectById(ctx context.Context, req *pb.GetProjectByIdRequest) (*pb.GetProjectByIdResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.GetProjectById(ctx, req)
		}
	}
	resp := &pb.GetProjectByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get project entity wrapper by name
func (s *projAuth) GetProjectWrapperByName(ctx context.Context, req *pb.GetProjectWrapperByNameRequest) (*pb.GetProjectWrapperByNameResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.GetProjectWrapperByName(ctx, req)
		}
	}
	resp := &pb.GetProjectWrapperByNameResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get project entity wrapper by id
func (s *projAuth) GetProjectWrapperById(ctx context.Context, req *pb.GetProjectWrapperByIdRequest) (*pb.GetProjectWrapperByIdResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.GetProjectWrapperById(ctx, req)
		}
	}
	resp := &pb.GetProjectWrapperByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// create a new status type
func (s *projAuth) CreateStatusType(ctx context.Context, req *pb.CreateStatusTypeRequest) (*pb.CreateStatusTypeResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.CreateStatusType(ctx, req)
		}
	}
	resp := &pb.CreateStatusTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// update a status type
func (s *projAuth) UpdateStatusType(ctx context.Context, req *pb.UpdateStatusTypeRequest) (*pb.UpdateStatusTypeResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.UpdateStatusType(ctx, req)
		}
	}
	resp := &pb.UpdateStatusTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// delete a status type
func (s *projAuth) DeleteStatusType(ctx context.Context, req *pb.DeleteStatusTypeRequest) (*pb.DeleteStatusTypeResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.DeleteStatusType(ctx, req)
		}
	}
	resp := &pb.DeleteStatusTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get status type by id
func (s *projAuth) GetStatusType(ctx context.Context, req *pb.GetStatusTypeRequest) (*pb.GetStatusTypeResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.GetStatusType(ctx, req)
		}
	}
	resp := &pb.GetStatusTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get all status types for this mservice id
func (s *projAuth) GetStatusTypes(ctx context.Context, req *pb.GetStatusTypesRequest) (*pb.GetStatusTypesResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.GetStatusTypes(ctx, req)
		}
	}
	resp := &pb.GetStatusTypesResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// create a new task
func (s *projAuth) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.CreateTask(ctx, req)
		}
	}
	resp := &pb.CreateTaskResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// update an existing task
func (s *projAuth) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.UpdateTaskResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.UpdateTask(ctx, req)
		}
	}
	resp := &pb.UpdateTaskResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// delete an existing task
func (s *projAuth) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*pb.DeleteTaskResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.DeleteTask(ctx, req)
		}
	}
	resp := &pb.DeleteTaskResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get a task by id
func (s *projAuth) GetTaskById(ctx context.Context, req *pb.GetTaskByIdRequest) (*pb.GetTaskByIdResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.GetTaskById(ctx, req)
		}
	}
	resp := &pb.GetTaskByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get a task with asspciations by id
func (s *projAuth) GetTaskWrapperById(ctx context.Context, req *pb.GetTaskWrapperByIdRequest) (*pb.GetTaskWrapperByIdResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.GetTaskWrapperById(ctx, req)
		}
	}
	resp := &pb.GetTaskWrapperByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// reorder the positions of child tasks
func (s *projAuth) ReorderChildTasks(ctx context.Context, req *pb.ReorderChildTasksRequest) (*pb.ReorderChildTasksResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.ReorderChildTasks(ctx, req)
		}
	}
	resp := &pb.ReorderChildTasksResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get list of tasks in project
func (s *projAuth) GetTasksByProject(ctx context.Context, req *pb.GetTasksByProjectRequest) (*pb.GetTasksByProjectResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.GetTasksByProject(ctx, req)
		}
	}
	resp := &pb.GetTasksByProjectResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// create a new team member for the project
func (s *projAuth) CreateTeamMember(ctx context.Context, req *pb.CreateTeamMemberRequest) (*pb.CreateTeamMemberResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.CreateTeamMember(ctx, req)
		}
	}
	resp := &pb.CreateTeamMemberResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// update an existing team member
func (s *projAuth) UpdateTeamMember(ctx context.Context, req *pb.UpdateTeamMemberRequest) (*pb.UpdateTeamMemberResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.UpdateTeamMember(ctx, req)
		}
	}
	resp := &pb.UpdateTeamMemberResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// delete an existing team member
func (s *projAuth) DeleteTeamMember(ctx context.Context, req *pb.DeleteTeamMemberRequest) (*pb.DeleteTeamMemberResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.DeleteTeamMember(ctx, req)
		}
	}
	resp := &pb.DeleteTeamMemberResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get team member by id
func (s *projAuth) GetTeamMemberById(ctx context.Context, req *pb.GetTeamMemberByIdRequest) (*pb.GetTeamMemberByIdResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.GetTeamMemberById(ctx, req)
		}
	}
	resp := &pb.GetTeamMemberByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get team members by project
func (s *projAuth) GetTeamMemberByProject(ctx context.Context, req *pb.GetTeamMemberByProjectRequest) (*pb.GetTeamMemberByProjectResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.GetTeamMemberByProject(ctx, req)
		}
	}
	resp := &pb.GetTeamMemberByProjectResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get team members by task
func (s *projAuth) GetTeamMemberByTask(ctx context.Context, req *pb.GetTeamMemberByTaskRequest) (*pb.GetTeamMemberByTaskResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.GetTeamMemberByTask(ctx, req)
		}
	}
	resp := &pb.GetTeamMemberByTaskResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// add a team member to a task
func (s *projAuth) AddTeamMemberToTask(ctx context.Context, req *pb.AddTeamMemberToTaskRequest) (*pb.AddTeamMemberToTaskResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.AddTeamMemberToTask(ctx, req)
		}
	}
	resp := &pb.AddTeamMemberToTaskResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// remove a team member from a task
func (s *projAuth) RemoveTeamMemberFromTask(ctx context.Context, req *pb.RemoveTeamMemberFromTaskRequest) (*pb.RemoveTeamMemberFromTaskResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.RemoveTeamMemberFromTask(ctx, req)
		}
	}
	resp := &pb.RemoveTeamMemberFromTaskResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// add to existing task hours for task and member
func (s *projAuth) AddTaskHours(ctx context.Context, req *pb.AddTaskHoursRequest) (*pb.AddTaskHoursResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.AddTaskHours(ctx, req)
		}
	}
	resp := &pb.AddTaskHoursResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// create a new project role type
func (s *projAuth) CreateProjectRoleType(ctx context.Context, req *pb.CreateProjectRoleTypeRequest) (*pb.CreateProjectRoleTypeResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.CreateProjectRoleType(ctx, req)
		}
	}
	resp := &pb.CreateProjectRoleTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// update an existing project role type
func (s *projAuth) UpdateProjectRoleType(ctx context.Context, req *pb.UpdateProjectRoleTypeRequest) (*pb.UpdateProjectRoleTypeResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.UpdateProjectRoleType(ctx, req)
		}
	}
	resp := &pb.UpdateProjectRoleTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// delete an existing project role type
func (s *projAuth) DeleteProjectRoleType(ctx context.Context, req *pb.DeleteProjectRoleTypeRequest) (*pb.DeleteProjectRoleTypeResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.DeleteProjectRoleType(ctx, req)
		}
	}
	resp := &pb.DeleteProjectRoleTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get a project role type by id
func (s *projAuth) GetProjectRoleType(ctx context.Context, req *pb.GetProjectRoleTypeRequest) (*pb.GetProjectRoleTypeResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.GetProjectRoleType(ctx, req)
		}
	}
	resp := &pb.GetProjectRoleTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}

// get all project role types for an mservice id
func (s *projAuth) GetProjectRoleTypes(ctx context.Context, req *pb.GetProjectRoleTypesRequest) (*pb.GetProjectRoleTypesResponse, error) {
	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.GetProjectRoleTypes(ctx, req)
		}
	}
	resp := &pb.GetProjectRoleTypesResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	return resp, err
}
