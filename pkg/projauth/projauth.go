// Copyright 2019-2021 Demian Harvill
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
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt"

	pb "github.com/gaterace/mproject/pkg/mserviceproject"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"crypto/rsa"
	"io/ioutil"
)

const (
	tokenExpiredMatch   = "Token is expired"
	tokenExpiredMessage = "token is expired"
)

var NotImplemented = errors.New("not implemented")

type ProjAuth struct {
	pb.UnimplementedMServiceProjectServer
	logger          log.Logger
	db              *sql.DB
	rsaPSSPublicKey *rsa.PublicKey
	projService     pb.MServiceProjectServer
}

// Get a new ProjAuth instance.
func NewProjectAuth(projService pb.MServiceProjectServer) *ProjAuth {
	svc := ProjAuth{}
	svc.projService = projService
	return &svc
}

// Set the logger for the ProjAuth instance.
func (s *ProjAuth) SetLogger(logger log.Logger) {
	s.logger = logger
}

// Set the database connection for the ProjAuth instance.
func (s *ProjAuth) SetDatabaseConnection(sqlDB *sql.DB) {
	s.db = sqlDB
}

// Set the public RSA key for the ProjAuth instance, used to validate JWT.
func (s *ProjAuth) SetPublicKey(publicKeyFile string) error {
	publicKey, err := ioutil.ReadFile(publicKeyFile)
	if err != nil {
		level.Error(s.logger).Log("what", "reading publicKeyFile", "error", err)
		return err
	}

	parsedKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKey)
	if err != nil {
		level.Error(s.logger).Log("what", "ParseRSAPublicKeyFromPEM", "error", err)
		return err
	}

	s.rsaPSSPublicKey = parsedKey
	return nil
}

// Bind our ProjAuth as the gRPC api server.
func (s *ProjAuth) NewApiServer(gServer *grpc.Server) error {
	if s != nil {
		pb.RegisterMServiceProjectServer(gServer, s)

	}
	return nil
}

// Get the JWT from the gRPC request context.
func (s *ProjAuth) GetJwtFromContext(ctx context.Context) (*map[string]interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("cannot get metadata from context")
	}

	tokens := md["token"]

	if (tokens == nil) || (len(tokens) == 0) {
		return nil, fmt.Errorf("cannot get token from context")
	}

	tokenString := tokens[0]

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
func (s *ProjAuth) CreateProject(ctx context.Context, req *pb.CreateProjectRequest) (*pb.CreateProjectResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.CreateProjectResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.CreateProject(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateProject",
		"project", req.GetName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// update an existing project
func (s *ProjAuth) UpdateProject(ctx context.Context, req *pb.UpdateProjectRequest) (*pb.UpdateProjectResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.UpdateProjectResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.UpdateProject(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateProject",
		"project", req.GetName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete an existing project
func (s *ProjAuth) DeleteProject(ctx context.Context, req *pb.DeleteProjectRequest) (*pb.DeleteProjectResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.DeleteProjectResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.DeleteProject(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteProject",
		"projectid", req.GetProjectId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get list of project names for this mservice id
func (s *ProjAuth) GetProjectNames(ctx context.Context, req *pb.GetProjectNamesRequest) (*pb.GetProjectNamesResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetProjectNamesResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.GetProjectNames(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetProjectNames",
		"mserviceid", req.GetMserviceId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get project entity by name
func (s *ProjAuth) GetProjectByName(ctx context.Context, req *pb.GetProjectByNameRequest) (*pb.GetProjectByNameResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetProjectByNameResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.GetProjectByName(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetProjectByName",
		"project", req.GetName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get project entity by id
func (s *ProjAuth) GetProjectById(ctx context.Context, req *pb.GetProjectByIdRequest) (*pb.GetProjectByIdResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetProjectByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.GetProjectById(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetProjectById",
		"projectid", req.GetProjectId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get project entity wrapper by name
func (s *ProjAuth) GetProjectWrapperByName(ctx context.Context, req *pb.GetProjectWrapperByNameRequest) (*pb.GetProjectWrapperByNameResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetProjectWrapperByNameResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.GetProjectWrapperByName(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetProjectWrapperByName",
		"project", req.GetName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get project entity wrapper by id
func (s *ProjAuth) GetProjectWrapperById(ctx context.Context, req *pb.GetProjectWrapperByIdRequest) (*pb.GetProjectWrapperByIdResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetProjectWrapperByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.GetProjectWrapperById(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetProjectWrapperById",
		"projectid", req.GetProjectId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// create a new status type
func (s *ProjAuth) CreateStatusType(ctx context.Context, req *pb.CreateStatusTypeRequest) (*pb.CreateStatusTypeResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.CreateStatusTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.CreateStatusType(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateStatusType",
		"statustype", req.GetStatusName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// update a status type
func (s *ProjAuth) UpdateStatusType(ctx context.Context, req *pb.UpdateStatusTypeRequest) (*pb.UpdateStatusTypeResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.UpdateStatusTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.UpdateStatusType(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateStatusType",
		"statustype", req.GetStatusName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete a status type
func (s *ProjAuth) DeleteStatusType(ctx context.Context, req *pb.DeleteStatusTypeRequest) (*pb.DeleteStatusTypeResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.DeleteStatusTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.DeleteStatusType(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteStatusType",
		"statusid", req.GetStatusId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get status type by id
func (s *ProjAuth) GetStatusType(ctx context.Context, req *pb.GetStatusTypeRequest) (*pb.GetStatusTypeResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetStatusTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.GetStatusType(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetStatusType",
		"statusid", req.GetStatusId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get all status types for this mservice id
func (s *ProjAuth) GetStatusTypes(ctx context.Context, req *pb.GetStatusTypesRequest) (*pb.GetStatusTypesResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetStatusTypesResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.GetStatusTypes(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetStatusTypes",
		"mserviceid", req.GetMserviceId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// create a new task
func (s *ProjAuth) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.CreateTaskResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.CreateTask(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateTask",
		"task", req.GetName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// update an existing task
func (s *ProjAuth) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.UpdateTaskResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.UpdateTaskResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.UpdateTask(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateTask",
		"task", req.GetName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete an existing task
func (s *ProjAuth) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*pb.DeleteTaskResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.DeleteTaskResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.DeleteTask(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteTask",
		"taskid", req.GetTaskId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get a task by id
func (s *ProjAuth) GetTaskById(ctx context.Context, req *pb.GetTaskByIdRequest) (*pb.GetTaskByIdResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetTaskByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.GetTaskById(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetTaskById",
		"taskid", req.GetTaskId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get a task with asspciations by id
func (s *ProjAuth) GetTaskWrapperById(ctx context.Context, req *pb.GetTaskWrapperByIdRequest) (*pb.GetTaskWrapperByIdResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetTaskWrapperByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			return s.projService.GetTaskWrapperById(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetTaskWrapperById",
		"taskid", req.GetTaskId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// reorder the positions of child tasks
func (s *ProjAuth) ReorderChildTasks(ctx context.Context, req *pb.ReorderChildTasksRequest) (*pb.ReorderChildTasksResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.ReorderChildTasksResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.ReorderChildTasks(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "ReorderChildTasks",
		"taskid", req.GetTaskId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get list of tasks in project
func (s *ProjAuth) GetTasksByProject(ctx context.Context, req *pb.GetTasksByProjectRequest) (*pb.GetTasksByProjectResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetTasksByProjectResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.GetTasksByProject(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetTasksByProject",
		"projectid", req.GetProjectId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// create a new team member for the project
func (s *ProjAuth) CreateTeamMember(ctx context.Context, req *pb.CreateTeamMemberRequest) (*pb.CreateTeamMemberResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.CreateTeamMemberResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.CreateTeamMember(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateTeamMember",
		"member", req.GetName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// update an existing team member
func (s *ProjAuth) UpdateTeamMember(ctx context.Context, req *pb.UpdateTeamMemberRequest) (*pb.UpdateTeamMemberResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.UpdateTeamMemberResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.UpdateTeamMember(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateTeamMember",
		"member", req.GetName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete an existing team member
func (s *ProjAuth) DeleteTeamMember(ctx context.Context, req *pb.DeleteTeamMemberRequest) (*pb.DeleteTeamMemberResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.DeleteTeamMemberResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.DeleteTeamMember(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteTeamMember",
		"memberid", req.GetMemberId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get team member by id
func (s *ProjAuth) GetTeamMemberById(ctx context.Context, req *pb.GetTeamMemberByIdRequest) (*pb.GetTeamMemberByIdResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetTeamMemberByIdResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.GetTeamMemberById(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetTeamMemberById",
		"memberid", req.GetMemberId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get team members by project
func (s *ProjAuth) GetTeamMemberByProject(ctx context.Context, req *pb.GetTeamMemberByProjectRequest) (*pb.GetTeamMemberByProjectResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetTeamMemberByProjectResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.GetTeamMemberByProject(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetTeamMemberByProject",
		"projectid", req.GetProjectId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get team members by task
func (s *ProjAuth) GetTeamMemberByTask(ctx context.Context, req *pb.GetTeamMemberByTaskRequest) (*pb.GetTeamMemberByTaskResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetTeamMemberByTaskResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") || (projsvc == "projro") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.GetTeamMemberByTask(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetTeamMemberByTask",
		"taskid", req.GetTaskId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// add a team member to a task
func (s *ProjAuth) AddTeamMemberToTask(ctx context.Context, req *pb.AddTeamMemberToTaskRequest) (*pb.AddTeamMemberToTaskResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.AddTeamMemberToTaskResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.AddTeamMemberToTask(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "AddTeamMemberToTask",
		"taskid", req.GetTaskId(),
		"memberid", req.GetMemberId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// remove a team member from a task
func (s *ProjAuth) RemoveTeamMemberFromTask(ctx context.Context, req *pb.RemoveTeamMemberFromTaskRequest) (*pb.RemoveTeamMemberFromTaskResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.RemoveTeamMemberFromTaskResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.RemoveTeamMemberFromTask(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "RemoveTeamMemberFromTask",
		"taskid", req.GetTaskId(),
		"memberid", req.GetMemberId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// add to existing task hours for task and member
func (s *ProjAuth) AddTaskHours(ctx context.Context, req *pb.AddTaskHoursRequest) (*pb.AddTaskHoursResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.AddTaskHoursResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if (projsvc == "projadmin") || (projsvc == "projrw") {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.AddTaskHours(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "AddTaskHours",
		"taskid", req.GetTaskId(),
		"memberid", req.GetMemberId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// create a new project role type
func (s *ProjAuth) CreateProjectRoleType(ctx context.Context, req *pb.CreateProjectRoleTypeRequest) (*pb.CreateProjectRoleTypeResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.CreateProjectRoleTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.CreateProjectRoleType(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "CreateProjectRoleType",
		"roletype", req.GetRoleName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// update an existing project role type
func (s *ProjAuth) UpdateProjectRoleType(ctx context.Context, req *pb.UpdateProjectRoleTypeRequest) (*pb.UpdateProjectRoleTypeResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.UpdateProjectRoleTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.UpdateProjectRoleType(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "UpdateProjectRoleType",
		"roletype", req.GetRoleName(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// delete an existing project role type
func (s *ProjAuth) DeleteProjectRoleType(ctx context.Context, req *pb.DeleteProjectRoleTypeRequest) (*pb.DeleteProjectRoleTypeResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.DeleteProjectRoleTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.DeleteProjectRoleType(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "DeleteProjectRoleType",
		"roleid", req.GetProjectRoleId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get a project role type by id
func (s *ProjAuth) GetProjectRoleType(ctx context.Context, req *pb.GetProjectRoleTypeRequest) (*pb.GetProjectRoleTypeResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetProjectRoleTypeResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.GetProjectRoleType(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetProjectRoleType",
		"roleid", req.GetProjectRoleId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get all project role types for an mservice id
func (s *ProjAuth) GetProjectRoleTypes(ctx context.Context, req *pb.GetProjectRoleTypesRequest) (*pb.GetProjectRoleTypesResponse, error) {
	start := time.Now().UnixNano()
	resp := &pb.GetProjectRoleTypesResponse{}
	resp.ErrorCode = 401
	resp.ErrorMessage = "not authorized"

	claims, err := s.GetJwtFromContext(ctx)
	if err == nil {
		projsvc := GetStringFromClaims(claims, "projsvc")
		if projsvc == "projadmin" {
			req.MserviceId = GetInt64FromClaims(claims, "aid")
			resp, err = s.projService.GetProjectRoleTypes(ctx, req)
		}
	} else {
		if err.Error() == tokenExpiredMatch {
			resp.ErrorCode = 498
			resp.ErrorMessage = tokenExpiredMessage
		}

		err = nil
	}

	duration := time.Now().UnixNano() - start
	level.Info(s.logger).Log("endpoint", "GetProjectRoleTypes",
		"mserviceid", req.GetMserviceId(),
		"errcode", resp.GetErrorCode(), "duration", duration)

	return resp, err
}

// get current server version and uptime - health check
func (s *ProjAuth) GetServerVersion(ctx context.Context, req *pb.GetServerVersionRequest) (*pb.GetServerVersionResponse, error) {
	return s.projService.GetServerVersion(ctx, req)
}
