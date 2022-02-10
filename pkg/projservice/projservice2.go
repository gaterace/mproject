// Copyright 2019-2022 Demian Harvill
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

package projservice

import (
	"context"
	"database/sql"
	"regexp"
	"strings"

	"github.com/go-kit/kit/log/level"

	"github.com/gaterace/dml-go/pkg/dml"

	_ "github.com/go-sql-driver/mysql"

	pb "github.com/gaterace/mproject/pkg/mserviceproject"
)

var nameValidator = regexp.MustCompile("^[a-z0-9_\\-]{1,32}$")

// create a new status type
func (s *projService) CreateStatusType(ctx context.Context, req *pb.CreateStatusTypeRequest) (*pb.CreateStatusTypeResponse, error) {
	resp := &pb.CreateStatusTypeResponse{}
	if !nameValidator.MatchString(req.GetStatusName()) {
		resp.ErrorCode = 510
		resp.ErrorMessage = "status_name invalid format"
		return resp, nil
	}

	desc := strings.TrimSpace(req.GetDescription())
	if desc == "" {
		resp.ErrorCode = 510
		resp.ErrorMessage = "description missing"
		return resp, nil
	}

	sqlstring := `INSERT INTO tb_StatusType 
		(intStatusId, dtmCreated, dtmModified, dtmDeleted, bitIsDeleted, intVersion, inbMserviceId, 
			chvStatusName, chvDescription) VALUES (?, NOW(), NOW(), NOW(), 0, 1, ?, ?, ?)`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	_, err = stmt.Exec(req.GetStatusId(), req.GetMserviceId(), req.GetStatusName(), desc)
	if err == nil {
		resp.Version = 1
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		level.Error(s.logger).Log("what", "Exec", "error", err)
		err = nil
	}

	return resp, err
}

// update a status type
func (s *projService) UpdateStatusType(ctx context.Context, req *pb.UpdateStatusTypeRequest) (*pb.UpdateStatusTypeResponse, error) {
	resp := &pb.UpdateStatusTypeResponse{}
	var err error

	sqlstring := `UPDATE tb_StatusType SET dtmModified = NOW(), intVersion = ?, chvStatusName = ?, chvDescription = ? 
	WHERE intStatusId = ? AND inbMserviceId = ? AND bitIsDeleted = 0 AND intVersion = ?`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetStatusName(), req.GetDescription(), req.GetStatusId(), req.GetMserviceId(), req.GetVersion())

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
		level.Error(s.logger).Log("what", "Exec", "error", err)
		err = nil
	}

	return resp, err
}

// delete a status type
func (s *projService) DeleteStatusType(ctx context.Context, req *pb.DeleteStatusTypeRequest) (*pb.DeleteStatusTypeResponse, error) {
	resp := &pb.DeleteStatusTypeResponse{}
	var err error

	sqlstring := `UPDATE tb_StatusType SET dtmDeleted = NOW(), bitIsDeleted = 1, intVersion = ? 
	WHERE intStatusId = ? AND inbMserviceId = ? AND bitIsDeleted = 0 AND intVersion = ?`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetStatusId(), req.GetMserviceId(), req.GetVersion())
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
		level.Error(s.logger).Log("what", "Exec", "error", err)
		err = nil
	}

	return resp, err
}

// get status type by id
func (s *projService) GetStatusType(ctx context.Context, req *pb.GetStatusTypeRequest) (*pb.GetStatusTypeResponse, error) {
	resp := &pb.GetStatusTypeResponse{}
	var err error

	sqlstring := `SELECT intStatusId, dtmCreated, dtmModified, intVersion, inbMserviceId, chvStatusName, chvDescription
	FROM tb_StatusType WHERE intStatusId = ? AND inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	var created string
	var modified string

	var statusType pb.StatusType

	err = stmt.QueryRow(req.GetStatusId(), req.GetMserviceId()).Scan(&statusType.StatusId, &created, &modified,
		&statusType.Version, &statusType.MserviceId, &statusType.StatusName,
		&statusType.Description)

	if err == nil {
		statusType.Created = dml.DateTimeFromString(created)
		statusType.Modified = dml.DateTimeFromString(modified)

		resp.ErrorCode = 0
		resp.StatusType = &statusType
	} else if err == sql.ErrNoRows {
		resp.ErrorCode = 404
		resp.ErrorMessage = "not found"
		err = nil
	} else {
		level.Error(s.logger).Log("what", "QueryRow", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		err = nil
	}

	return resp, err
}

// get all status types for this mservice id
func (s *projService) GetStatusTypes(ctx context.Context, req *pb.GetStatusTypesRequest) (*pb.GetStatusTypesResponse, error) {
	resp := &pb.GetStatusTypesResponse{}
	var err error

	sqlstring := `SELECT intStatusId, dtmCreated, dtmModified, intVersion, inbMserviceId, chvStatusName, chvDescription
	FROM tb_StatusType WHERE inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(req.GetMserviceId())

	if err != nil {
		level.Error(s.logger).Log("what", "Query", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	defer rows.Close()
	for rows.Next() {
		var created string
		var modified string
		var statusType pb.StatusType

		err := rows.Scan(&statusType.StatusId, &created, &modified, &statusType.Version, &statusType.MserviceId,
			&statusType.StatusName, &statusType.Description)

		if err != nil {
			level.Error(s.logger).Log("what", "Scan", "error", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		statusType.Created = dml.DateTimeFromString(created)
		statusType.Modified = dml.DateTimeFromString(modified)
		resp.StatusTypes = append(resp.StatusTypes, &statusType)
	}

	return resp, err
}

// create a new task
func (s *projService) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	resp := &pb.CreateTaskResponse{}

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

	// make sure project id is valid
	sqlstring1 := `SELECT inbProjectId FROM tb_Project WHERE inbProjectId = ? AND inbMserviceId = ? AND bitIsDeleted = 0`
	stmt1, err := s.db.Prepare(sqlstring1)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt1.Close()

	var projectId int64

	err = stmt1.QueryRow(req.GetProjectId(), req.GetMserviceId()).Scan(&projectId)
	if err == nil {
		// OK
	} else if err == sql.ErrNoRows {
		resp.ErrorCode = 404
		resp.ErrorMessage = "project for task not found"
		return resp, nil
	} else {
		level.Error(s.logger).Log("what", "QueryRow", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		return resp, nil
	}

	if req.GetParentId() != 0 {

		// make sure parent task id is valid
		sqlstring2 := `SELECT inbTaskId FROM tb_Task WHERE inbTaskId = ? AND inbProjectId = ? AND bitIsDeleted = 0`
		stmt2, err := s.db.Prepare(sqlstring2)
		if err != nil {
			level.Error(s.logger).Log("what", "Prepare", "error", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = "db.Prepare failed"
			return resp, nil
		}

		defer stmt2.Close()

		var parentTaskId int64
		err = stmt2.QueryRow(req.GetParentId(), req.GetProjectId()).Scan(&parentTaskId)
		if err == nil {
			// OK
		} else if err == sql.ErrNoRows {
			resp.ErrorCode = 404
			resp.ErrorMessage = "parent task not found"
			return resp, nil
		} else {
			level.Error(s.logger).Log("what", "QueryRow", "error", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

	}

	sqlstring := `INSERT INTO tb_Task (dtmCreated, dtmModified, dtmDeleted, bitIsDeleted, intVersion, inbMserviceId, 
		inbProjectId, chvName, chvDescription, intStatusId, dtmStartDate, dtmEndDate, intPriority, inbParentId,
		intPosition) VALUES (NOW(), NOW(), NOW(), 0, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	start_date := req.GetStartDate().TimeFromDateTime()
	end_date := req.GetEndDate().TimeFromDateTime()

	res, err := stmt.Exec(req.GetMserviceId(), req.GetProjectId(), req.GetName(), req.GetDescription(), req.GetStatusId(), start_date,
		end_date, req.GetPriority(), req.GetParentId(), req.GetPosition())

	if err == nil {
		taskId, err := res.LastInsertId()
		if err != nil {
			level.Error(s.logger).Log("what", "LastInsertId", "error", err)
		} else {
			level.Debug(s.logger).Log("taskId", taskId)
		}

		resp.TaskId = taskId
		resp.Version = 1
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		level.Error(s.logger).Log("what", "Exec", "error", err)
		err = nil
	}

	return resp, err
}

// update an existing task
func (s *projService) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.UpdateTaskResponse, error) {
	resp := &pb.UpdateTaskResponse{}
	var err error

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

	sqlstring := `UPDATE tb_Task SET dtmModified = NOW(), intVersion = ?, chvName = ?, chvDescription = ?, intStatusId = ?, 
	dtmStartDate = ?, dtmEndDate = ?, intPriority = ?, intPosition = ? WHERE inbTaskId = ? AND intVersion = ? 
	AND inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	start_date := req.GetStartDate().TimeFromDateTime()
	end_date := req.GetEndDate().TimeFromDateTime()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetName(), req.GetDescription(), req.GetStatusId(), start_date,
		end_date, req.GetPriority(), req.GetPosition(), req.GetTaskId(), req.GetVersion(), req.GetMserviceId())

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
		level.Error(s.logger).Log("what", "Exec", "error", err)
		err = nil
	}

	return resp, err
}

// delete an existing task
func (s *projService) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*pb.DeleteTaskResponse, error) {
	resp := &pb.DeleteTaskResponse{}
	var err error

	sqlstring := `UPDATE tb_Task SET dtmDeleted = NOW(), intVersion = ?, bitIsDeleted = 1
	 WHERE inbTaskId = ? AND intVersion = ? AND inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetTaskId(), req.GetVersion(), req.GetMserviceId())
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
		level.Error(s.logger).Log("what", "Exec", "error", err)
		err = nil
	}

	return resp, err
}

// get a task by id
func (s *projService) GetTaskById(ctx context.Context, req *pb.GetTaskByIdRequest) (*pb.GetTaskByIdResponse, error) {
	resp := &pb.GetTaskByIdResponse{}

	var err error

	sqlstring := `SELECT t.inbTaskId, t.dtmCreated, t.dtmModified, t.intVersion, t.inbMserviceId, t.inbProjectId, t.chvName,
	t.chvDescription, t.intStatusId, t.dtmStartDate, t.dtmEndDate, t.intPriority, t.inbParentId, t.intPosition, s.chvStatusName 
	FROM tb_Task AS t 
	JOIN tb_StatusType AS s ON t.intStatusId = s.intStatusId 
	WHERE t.inbTaskId = ? AND t.inbMserviceId = ? AND t.bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	var created string
	var modified string
	var start_date string
	var end_date string

	var task pb.Task

	err = stmt.QueryRow(req.GetTaskId(), req.GetMserviceId()).Scan(&task.TaskId, &created, &modified, &task.Version, &task.MserviceId,
		&task.ProjectId, &task.Name, &task.Description, &task.StatusId, &start_date, &end_date, &task.Priority, &task.ParentId,
		&task.Position, &task.StatusName)

	if err == nil {
		task.Created = dml.DateTimeFromString(created)
		task.Modified = dml.DateTimeFromString(modified)
		task.StartDate = dml.DateTimeFromString(start_date)
		task.EndDate = dml.DateTimeFromString(end_date)
		resp.ErrorCode = 0
		resp.Task = &task
	} else if err == sql.ErrNoRows {
		resp.ErrorCode = 404
		resp.ErrorMessage = "not found"
		err = nil
	} else {
		level.Error(s.logger).Log("what", "QueryRow", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()
		err = nil
	}

	return resp, err
}

// get a task with associations by id
func (s *projService) GetTaskWrapperById(ctx context.Context, req *pb.GetTaskWrapperByIdRequest) (*pb.GetTaskWrapperByIdResponse, error) {
	resp := &pb.GetTaskWrapperByIdResponse{}
	var err error

	sqlstring1 := `SELECT inbProjectId FROM tb_Task WHERE inbTaskId = ? AND inbMserviceId = ? AND bitIsDeleted = 0`
	stmt1, err := s.db.Prepare(sqlstring1)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt1.Close()

	var existingProjectId int64
	err = stmt1.QueryRow(req.GetTaskId(), req.GetMserviceId()).Scan(&existingProjectId)
	if err != nil {
		resp.ErrorCode = 404
		resp.ErrorMessage = "referenced task not found"
		return resp, nil
	}

	gResp, wraps := s.GetProjectTaskWrapperHelper(existingProjectId, req.GetMserviceId())
	if gResp.ErrorCode != 0 {
		resp.ErrorCode = gResp.ErrorCode
		resp.ErrorMessage = gResp.ErrorMessage
		return resp, nil
	}

	for _, wrap := range wraps {
		if wrap.GetTaskId() == req.GetTaskId() {
			resp.TaskWrapper = wrap
			break
		}
	}

	return resp, err
}

// reorder the positions of child tasks
func (s *projService) ReorderChildTasks(ctx context.Context, req *pb.ReorderChildTasksRequest) (*pb.ReorderChildTasksResponse, error) {
	resp := &pb.ReorderChildTasksResponse{}
	var err error

	sqlstring := `UPDATE tb_Task SET dtmModified = NOW(), intVersion = ?
	 WHERE inbTaskId = ? AND intVersion = ? AND inbMserviceId = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetTaskId(), req.GetVersion(), req.GetMserviceId())
	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			resp.Version = req.GetVersion() + 1
		} else {
			resp.ErrorCode = 404
			resp.ErrorMessage = "not found"
			return resp, nil
		}
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		level.Error(s.logger).Log("what", "Exec", "error", err)
		return resp, nil
	}

	sqlstring1 := `UPDATE tb_Task SET dtmModified = NOW(), intVersion = intVersion + 1, intPosition = ?
	WHERE inbTaskId = ? AND inbMserviceId = ? AND bitIsDeleted = 0 AND inbParentId = ?`

	stmt1, err := s.db.Prepare(sqlstring1)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt1.Close()

	for pos, childId := range req.GetChildTaskIds() {
		// s.logger.Printf("childId: %d, taskId: %d, mservice: %d, pos: %d\n", childId, req.GetMserviceId(), req.GetTaskId(), pos)
		res, err := stmt1.Exec(pos+1, childId, req.GetMserviceId(), req.GetTaskId())
		if err == nil {
			rowsAffected, _ := res.RowsAffected()
			if rowsAffected != 1 {
				resp.ErrorCode = 404
				resp.ErrorMessage = "not found"
				return resp, nil
			}
		} else {
			resp.ErrorCode = 501
			resp.ErrorMessage = err.Error()
			level.Error(s.logger).Log("what", "Exec", "error", err)
			return resp, nil
		}
	}

	return resp, err
}

// get list of tasks in project
func (s *projService) GetTasksByProject(ctx context.Context, req *pb.GetTasksByProjectRequest) (*pb.GetTasksByProjectResponse, error) {
	resp := &pb.GetTasksByProjectResponse{}
	var err error

	gResp, tasks := s.GetProjectTasksHelper(req.GetProjectId(), req.GetMserviceId())

	resp.ErrorCode = gResp.ErrorCode
	resp.ErrorMessage = gResp.ErrorMessage
	if gResp.ErrorCode == 0 {
		resp.Tasks = tasks
	}

	return resp, err
}
