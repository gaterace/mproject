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

// Package  projservice frovides the implemantation for the MServiceProject gRPC service.
package projservice

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	pb "github.com/gaterace/mproject/pkg/mserviceproject"
	"google.golang.org/grpc"
)

var NotImplemented = errors.New("not implemented")

type projService struct {
	logger *log.Logger
	db     *sql.DB
	startSecs int64
}

// Get a new projService instance.
func NewProjectService() *projService {
	svc := projService{}
	svc.startSecs = time.Now().Unix()
	return &svc
}

// Set the logger for the projService instance.
func (s *projService) SetLogger(logger *log.Logger) {
	s.logger = logger
}

// Set the database connection for the projService instance.
func (s *projService) SetDatabaseConnection(sqlDB *sql.DB) {
	s.db = sqlDB
}

// Bind this projService the gRPC server api.
func (s *projService) NewApiServer(gServer *grpc.Server) error {
	if s != nil {
		pb.RegisterMServiceProjectServer(gServer, s)

	}
	return nil
}

// create a new project
func (s *projService) CreateProject(ctx context.Context, req *pb.CreateProjectRequest) (*pb.CreateProjectResponse, error) {
	s.logger.Printf("CreateProject called, name: %s, desc: %s, status_id: %d\n", req.GetName(), req.GetDescription(), req.GetStatusId())
	resp := &pb.CreateProjectResponse{}
	if !nameValidator.MatchString(req.GetName()) {
		resp.ErrorCode = 510
		resp.ErrorMessage = "name invalid format"
		return resp, nil
	}

	desc := strings.TrimSpace(req.GetDescription())
	if desc == "" {
		resp.ErrorCode = 510
		resp.ErrorMessage = "description missing"
		return resp, nil
	}

	sqlstring := `INSERT INTO tb_Project
	(dtmCreated, dtmModified, dtmDeleted, bitIsDeleted, intVersion, inbMserviceId, chvName, chvDescription,
		intStatusId, dtmStartDate, dtmEndDate) VALUES(NOW(), NOW(), NOW(), 0, 1, ?, ?, ?, ?, ?, ?)`
	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	start_date := req.GetStartDate().TimeFromDateTime()
	end_date := req.GetEndDate().TimeFromDateTime()

	res, err := stmt.Exec(req.GetMserviceId(), req.GetName(), desc, req.GetStatusId(), start_date, end_date)
	if err == nil {
		projectId, err := res.LastInsertId()
		if err != nil {
			s.logger.Printf("LastInsertId err: %v\n", err)
		} else {
			s.logger.Printf("projectId: %d", projectId)
		}

		resp.ProjectId = projectId
		resp.Version = 1
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, err

}

// update an existing project
func (s *projService) UpdateProject(ctx context.Context, req *pb.UpdateProjectRequest) (*pb.UpdateProjectResponse, error) {
	s.logger.Printf("UpdateProject called, pid: %d, name: %s, desc: %s, status_id: %d\n", req.GetProjectId(), req.GetName(), req.GetDescription(), req.GetStatusId())
	resp := &pb.UpdateProjectResponse{}
	if !nameValidator.MatchString(req.GetName()) {
		resp.ErrorCode = 510
		resp.ErrorMessage = "name invalid format"
		return resp, nil
	}

	desc := strings.TrimSpace(req.GetDescription())
	if desc == "" {
		resp.ErrorCode = 510
		resp.ErrorMessage = "description missing"
		return resp, nil
	}

	sqlstring := `UPDATE tb_Project SET dtmModified = NOW(), intVersion = ?, chvName = ?, chvDescription = ?, intStatusId = ?, 
	dtmStartDate = ?, dtmEndDate = ?  WHERE inbProjectId = ? AND intVersion = ? AND inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	start_date := req.GetStartDate().TimeFromDateTime()
	end_date := req.GetEndDate().TimeFromDateTime()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetName(), req.GetDescription(), req.GetStatusId(), start_date, end_date, req.GetProjectId(), req.GetVersion(), req.GetMserviceId())
	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = req.GetVersion() + 1
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, err

}

// delete an existing project
func (s *projService) DeleteProject(ctx context.Context, req *pb.DeleteProjectRequest) (*pb.DeleteProjectResponse, error) {
	s.logger.Printf("DeleteProject called, pid: %d, version: %d\n", req.GetProjectId(), req.GetVersion())
	resp := &pb.DeleteProjectResponse{}

	sqlstring := `UPDATE tb_Project SET dtmDeleted = NOW(), intVersion = ?, bitIsDeleted = 1
	 WHERE inbProjectId = ? AND intVersion = ? AND inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetProjectId(), req.GetVersion(), req.GetMserviceId())
	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = req.GetVersion() + 1
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		s.logger.Printf("err: %v\n", err)
		err = nil
	}

	return resp, err

}

// get list of project names for this mservice id
func (s *projService) GetProjectNames(ctx context.Context, req *pb.GetProjectNamesRequest) (*pb.GetProjectNamesResponse, error) {
	s.logger.Printf("GetProjectByName called, mservice: %d\n", req.GetMserviceId())
	resp := &pb.GetProjectNamesResponse{}
	var err error

	sqlstring := `SELECT chvName FROM tb_Project WHERE inbMserviceId = ? AND  bitIsDeleted = 0`
	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		s.logger.Printf("db.Prepare sqlstring failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(req.GetMserviceId())

	if err != nil {
		s.logger.Printf("query failed: %v\n", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	defer rows.Close()
	for rows.Next() {
		var projectName string
		err = rows.Scan(&projectName)

		if err != nil {
			s.logger.Printf("query rows scan  failed: %v\n", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		resp.Names = append(resp.Names, projectName)
	}

	return resp, err
}

// get project entity by name
func (s *projService) GetProjectByName(ctx context.Context, req *pb.GetProjectByNameRequest) (*pb.GetProjectByNameResponse, error) {
	s.logger.Printf("GetProjectByName called, name: %s\n", req.GetName())
	resp := &pb.GetProjectByNameResponse{}
	var err error

	gResp, project := s.GetProjectByNameHelper(req.GetName(), req.GetMserviceId())
	resp.ErrorCode = gResp.ErrorCode
	resp.ErrorMessage = gResp.ErrorMessage
	if gResp.ErrorCode == 0 {
		resp.Project = project
	}

	return resp, err
}

// get project entity by id
func (s *projService) GetProjectById(ctx context.Context, req *pb.GetProjectByIdRequest) (*pb.GetProjectByIdResponse, error) {
	s.logger.Printf("GetProjectById called, project_id: %d\n", req.GetProjectId())
	resp := &pb.GetProjectByIdResponse{}
	var err error

	gResp, project := s.GetProjectByIdHelper(req.GetProjectId(), req.GetMserviceId())
	resp.ErrorCode = gResp.ErrorCode
	resp.ErrorMessage = gResp.ErrorMessage
	if gResp.ErrorCode == 0 {
		resp.Project = project
	}

	return resp, err
}

// get project entity wrapper by name
func (s *projService) GetProjectWrapperByName(ctx context.Context, req *pb.GetProjectWrapperByNameRequest) (*pb.GetProjectWrapperByNameResponse, error) {
	s.logger.Printf("GetProjectWrapperByName called, name: %s\n", req.GetName())
	resp := &pb.GetProjectWrapperByNameResponse{}
	var err error

	gResp, project := s.GetProjectByNameHelper(req.GetName(), req.GetMserviceId())
	resp.ErrorCode = gResp.ErrorCode
	resp.ErrorMessage = gResp.ErrorMessage
	if gResp.ErrorCode != 0 {
		return resp, nil
	}

	wrap := convertProjectToWrapper(project)

	projectId := wrap.GetProjectId()

	gResp, members := s.GetTeamMembersHelper(projectId, req.GetMserviceId())
	if gResp.ErrorCode == 0 {
		wrap.TeamMembers = members
	}

	gResp, wraps := s.GetProjectTaskWrapperHelper(projectId, req.GetMserviceId())

	if gResp.ErrorCode != 0 {
		resp.ErrorCode = gResp.ErrorCode
		resp.ErrorMessage = gResp.ErrorMessage
		return resp, nil
	}

	for _, wtask := range wraps {
		if wtask.GetParentId() == 0 {
			wrap.ChildTaskWrappers = append(wrap.ChildTaskWrappers, wtask)
		}
	}

	resp.ProjectWrapper = wrap

	return resp, err
}

// get project entity wrapper by id
func (s *projService) GetProjectWrapperById(ctx context.Context, req *pb.GetProjectWrapperByIdRequest) (*pb.GetProjectWrapperByIdResponse, error) {
	s.logger.Printf("GetProjectWrapperById called, project_id: %d\n", req.GetProjectId())
	resp := &pb.GetProjectWrapperByIdResponse{}
	var err error

	gResp, project := s.GetProjectByIdHelper(req.GetProjectId(), req.GetMserviceId())
	resp.ErrorCode = gResp.ErrorCode
	resp.ErrorMessage = gResp.ErrorMessage
	if gResp.ErrorCode != 0 {
		return resp, nil
	}

	wrap := convertProjectToWrapper(project)

	projectId := wrap.GetProjectId()

	gResp, members := s.GetTeamMembersHelper(projectId, req.GetMserviceId())
	if gResp.ErrorCode == 0 {
		wrap.TeamMembers = members
	}

	gResp, wraps := s.GetProjectTaskWrapperHelper(projectId, req.GetMserviceId())

	if gResp.ErrorCode != 0 {
		resp.ErrorCode = gResp.ErrorCode
		resp.ErrorMessage = gResp.ErrorMessage
		return resp, nil
	}

	for _, wtask := range wraps {
		if wtask.GetParentId() == 0 {
			wrap.ChildTaskWrappers = append(wrap.ChildTaskWrappers, wtask)
		}
	}

	resp.ProjectWrapper = wrap

	return resp, err
}
