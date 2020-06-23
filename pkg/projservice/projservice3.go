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

package projservice

import (
	"context"
	"database/sql"
	"github.com/go-kit/kit/log/level"
	"strings"
	"time"

	"github.com/gaterace/dml-go/pkg/dml"

	_ "github.com/go-sql-driver/mysql"

	pb "github.com/gaterace/mproject/pkg/mserviceproject"
)

// create a new team member for the project
func (s *projService) CreateTeamMember(ctx context.Context, req *pb.CreateTeamMemberRequest) (*pb.CreateTeamMemberResponse, error) {
	resp := &pb.CreateTeamMemberResponse{}
	var err error

	sqlstring := `INSERT INTO tb_TeamMember (dtmCreated, dtmModified, dtmDeleted, bitIsDeleted, intVersion, inbMserviceId, inbProjectId,
		chvName, intProjectRoleId, chvEmail) VALUES(NOW(), NOW(), NOW(), 0, 1, ?, ?, ?, ?, ?)`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetMserviceId(), req.GetProjectId(), req.GetName(), req.GetProjectRoleId(), req.GetEmail())
	if err == nil {
		memberId, err := res.LastInsertId()
		if err != nil {
			level.Error(s.logger).Log("what", "LastInsertId", "error", err)
		} else {
			level.Debug(s.logger).Log("memberId", memberId)
		}

		resp.MemberId = memberId
		resp.Version = 1
	} else {
		resp.ErrorCode = 501
		resp.ErrorMessage = err.Error()
		level.Error(s.logger).Log("what", "Exec", "error", err)
		err = nil
	}

	return resp, err
}

// update an existing team member
func (s *projService) UpdateTeamMember(ctx context.Context, req *pb.UpdateTeamMemberRequest) (*pb.UpdateTeamMemberResponse, error) {
	resp := &pb.UpdateTeamMemberResponse{}
	var err error

	sqlstring := `UPDATE tb_TeamMember SET dtmModified = NOW(), intVersion = ?, chvName = ?, intProjectRoleId = ?, chvEmail = ?
	WHERE inbMemberId = ? AND inbMserviceId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetName(), req.GetProjectRoleId(), req.GetEmail(), req.GetMemberId(),
		req.GetMserviceId(), req.GetVersion())

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

// delete an existing team member
func (s *projService) DeleteTeamMember(ctx context.Context, req *pb.DeleteTeamMemberRequest) (*pb.DeleteTeamMemberResponse, error) {
	resp := &pb.DeleteTeamMemberResponse{}
	var err error

	sqlstring := `UPDATE tb_TeamMember SET dtmDeleted = NOW(), intVersion = ?, bitIsDeleted = 1
	WHERE inbMemberId = ? AND inbMserviceId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetMemberId(), req.GetMserviceId(), req.GetVersion())
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

// get team member by id
func (s *projService) GetTeamMemberById(ctx context.Context, req *pb.GetTeamMemberByIdRequest) (*pb.GetTeamMemberByIdResponse, error) {
	resp := &pb.GetTeamMemberByIdResponse{}
	var err error

	sqlstring := `SELECT m.inbMemberId, m.dtmCreated, m.dtmModified, m.intVersion, m.inbMserviceId, m.inbProjectId, m.chvName,
	m.intProjectRoleId, m.chvEmail, r.chvRoleName
	FROM tb_TeamMember AS m
	JOIN tb_ProjectRoleType AS r ON m.intProjectRoleId = r.intProjectRoleId
	WHERE m.inbMemberId = ? AND m.inbMserviceId = ? AND m.bitIsDeleted = 0`

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
	var member pb.TeamMember

	err = stmt.QueryRow(req.GetMemberId(), req.GetMserviceId()).Scan(&member.MemberId, &created, &modified, &member.Version,
		&member.MserviceId, &member.ProjectId, &member.Name, &member.ProjectRoleId, &member.Email, &member.RoleName)

	if err == nil {
		member.Created = dml.DateTimeFromString(created)
		member.Modified = dml.DateTimeFromString(modified)
		resp.ErrorCode = 0
		resp.TeamMember = &member
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

// get team members by project
func (s *projService) GetTeamMemberByProject(ctx context.Context, req *pb.GetTeamMemberByProjectRequest) (*pb.GetTeamMemberByProjectResponse, error) {
	resp := &pb.GetTeamMemberByProjectResponse{}
	var err error

	gResp, members := s.GetTeamMembersHelper(req.GetProjectId(), req.GetMserviceId())
	resp.ErrorCode = gResp.ErrorCode
	resp.ErrorMessage = gResp.ErrorMessage
	if gResp.ErrorCode == 0 {
		resp.TeamMembers = members
	}

	return resp, err
}

// get team members by task
func (s *projService) GetTeamMemberByTask(ctx context.Context, req *pb.GetTeamMemberByTaskRequest) (*pb.GetTeamMemberByTaskResponse, error) {
	resp := &pb.GetTeamMemberByTaskResponse{}
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

	sqlstring := `SELECT m.inbMemberId,  m.dtmCreated, m.dtmModified, m.intVersion,
	m.inbMserviceId, m.inbProjectId, m.chvName, m.intProjectRoleId, m.chvEmail, t.decTaskHours, r.chvRoleName 
	FROM tb_TaskToMember AS t 
	JOIN tb_TeamMember AS m ON t.inbMemberId = m.inbMemberId
	JOIN tb_ProjectRoleType AS r ON m.intProjectRoleId = r.intProjectRoleId
	WHERE t.inbProjectId = ? AND t.inbTaskId= ? AND t.inbMserviceId = ?
	AND t.bitIsDeleted = 0 AND m.bitIsDeleted = 0`
	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()
	rows, err := stmt.Query(existingProjectId, req.GetTaskId(), req.GetMserviceId())
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
		var task_hours string
		var member pb.TeamMember

		err := rows.Scan(&member.MemberId, &created, &modified, &member.Version, &member.MserviceId, &member.ProjectId,
			&member.Name, &member.ProjectRoleId, &member.Email, &task_hours, &member.RoleName)

		if err != nil {
			level.Error(s.logger).Log("what", "Scan", "error", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		member.Created = dml.DateTimeFromString(created)
		member.Modified = dml.DateTimeFromString(modified)
		d, err := dml.DecimalFromString(task_hours)
		if err == nil {
			member.TaskHours = d
		}

		resp.TeamMembers = append(resp.TeamMembers, &member)
	}

	return resp, err
}

// add a team member to a task
func (s *projService) AddTeamMemberToTask(ctx context.Context, req *pb.AddTeamMemberToTaskRequest) (*pb.AddTeamMemberToTaskResponse, error) {
	resp := &pb.AddTeamMemberToTaskResponse{}
	var err error

	// make sure that refered task is in this mservice id
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

	// make sure that refered team member is in this mservice id
	sqlstring2 := `SELECT inbMemberId FROM tb_TeamMember WHERE inbMemberId = ? AND inbMserviceId = ? AND bitIsDeleted = 0`
	stmt2, err := s.db.Prepare(sqlstring2)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt2.Close()

	var existingMemberId int64

	err = stmt2.QueryRow(req.GetMemberId(), req.GetMserviceId()).Scan(&existingMemberId)

	if err != nil {
		resp.ErrorCode = 404
		resp.ErrorMessage = "referenced member not found"
		return resp, nil
	}

	sqlstring3 := `INSERT INTO tb_TaskToMember
	(inbProjectId, inbTaskId, inbMemberId, dtmCreated, dtmModified, dtmDeleted, bitIsDeleted, inbMserviceId, decTaskHours)
	VALUES (?, ?, ?, NOW(), NOW(), NOW(), 0, ?, 0.0)`
	stmt3, err := s.db.Prepare(sqlstring3)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt3.Close()

	_, err = stmt3.Exec(existingProjectId, req.GetTaskId(), req.GetMemberId(), req.GetMserviceId())
	if err == nil {
		// OK
		return resp, nil
	}

	// might have previously deleted, reuse
	sqlstring4 := `UPDATE tb_TaskToMember SET dtmCreated = NOW(), dtmModified = NOW(), dtmDeleted = NOW(),
	bitIsDeleted = 0, decTaskHours = 0.0 WHERE inbProjectId = ? AND inbTaskId = ? AND inbMemberId = ? AND inbMserviceId = ?
	AND bitIsDeleted = 1`

	stmt4, err := s.db.Prepare(sqlstring4)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt4.Close()

	res, err := stmt4.Exec(existingProjectId, req.GetTaskId(), req.GetMemberId(), req.GetMserviceId())

	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected != 1 {
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

// remove a team member from a task
func (s *projService) RemoveTeamMemberFromTask(ctx context.Context, req *pb.RemoveTeamMemberFromTaskRequest) (*pb.RemoveTeamMemberFromTaskResponse, error) {
	resp := &pb.RemoveTeamMemberFromTaskResponse{}
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

	sqlstring := `UPDATE tb_TaskToMember SET dtmDeleted = NOW(),
	bitIsDeleted = 1, decTaskHours = 0.0 WHERE inbProjectId = ? AND inbTaskId = ? AND inbMemberId = ? AND inbMserviceId = ?
	AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(existingProjectId, req.GetTaskId(), req.GetMemberId(), req.GetMserviceId())

	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected != 1 {
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

// add to existing task hours for task and member
func (s *projService) AddTaskHours(ctx context.Context, req *pb.AddTaskHoursRequest) (*pb.AddTaskHoursResponse, error) {
	resp := &pb.AddTaskHoursResponse{}
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

	sqlstring := `UPDATE tb_TaskToMember SET dtmModified = NOW(),
	decTaskHours = decTaskHours + ? WHERE inbProjectId = ? AND inbTaskId = ? AND inbMemberId = ? AND inbMserviceId = ?
	AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetTaskHours().StringFromDecimal(), existingProjectId, req.GetTaskId(), req.GetMemberId(), req.GetMserviceId())

	if err == nil {
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected != 1 {
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

// create a new project role type
func (s *projService) CreateProjectRoleType(ctx context.Context, req *pb.CreateProjectRoleTypeRequest) (*pb.CreateProjectRoleTypeResponse, error) {
	resp := &pb.CreateProjectRoleTypeResponse{}
	if !nameValidator.MatchString(req.GetRoleName()) {
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

	sqlstring := `INSERT INTO tb_ProjectRoleType
	(intProjectRoleId, dtmCreated, dtmModified, dtmDeleted, bitIsDeleted, intVersion, inbMserviceId, chvRoleName, chvDescription)
	VALUES(?, NOW(), NOW(), NOW(), 0, 1, ?, ?, ?)`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	_, err = stmt.Exec(req.GetProjectRoleId(), req.GetMserviceId(), req.GetRoleName(), desc)
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

// update an existing project role type
func (s *projService) UpdateProjectRoleType(ctx context.Context, req *pb.UpdateProjectRoleTypeRequest) (*pb.UpdateProjectRoleTypeResponse, error) {
	resp := &pb.UpdateProjectRoleTypeResponse{}
	var err error

	sqlstring := `UPDATE tb_ProjectRoleType SET dtmModified = NOW(), intVersion = ?, chvRoleName = ?, chvDescription = ?
	WHERE  intProjectRoleId = ? AND inbMserviceId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetRoleName(), req.GetDescription(), req.GetProjectRoleId(),
		req.GetMserviceId(), req.GetVersion())

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

// delete an existing project role type
func (s *projService) DeleteProjectRoleType(ctx context.Context, req *pb.DeleteProjectRoleTypeRequest) (*pb.DeleteProjectRoleTypeResponse, error) {
	resp := &pb.DeleteProjectRoleTypeResponse{}
	var err error

	sqlstring := `UPDATE tb_ProjectRoleType SET dtmDeleted = NOW(), intVersion = ?, bitIsDeleted = 1
	WHERE  intProjectRoleId = ? AND inbMserviceId = ? AND intVersion = ? AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	res, err := stmt.Exec(req.GetVersion()+1, req.GetProjectRoleId(), req.GetMserviceId(), req.GetVersion())

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

// get a project role type by id
func (s *projService) GetProjectRoleType(ctx context.Context, req *pb.GetProjectRoleTypeRequest) (*pb.GetProjectRoleTypeResponse, error) {
	resp := &pb.GetProjectRoleTypeResponse{}
	var err error

	sqlstring := `SELECT intProjectRoleId, dtmCreated, dtmModified, intVersion, inbMserviceId, chvRoleName, chvDescription
	FROM tb_ProjectRoleType WHERE intProjectRoleId = ? AND inbMserviceId = ? AND bitIsDeleted = 0`

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
	var role pb.ProjectRoleType

	err = stmt.QueryRow(req.GetProjectRoleId(), req.GetMserviceId()).Scan(&role.ProjectRoleId, &created, &modified, &role.Version,
		&role.MserviceId, &role.RoleName, &role.Description)

	if err == nil {
		role.Created = dml.DateTimeFromString(created)
		role.Modified = dml.DateTimeFromString(modified)

		resp.ErrorCode = 0
		resp.RoleType = &role
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

// get all project role types for an mservice id
func (s *projService) GetProjectRoleTypes(ctx context.Context, req *pb.GetProjectRoleTypesRequest) (*pb.GetProjectRoleTypesResponse, error) {
	resp := &pb.GetProjectRoleTypesResponse{}
	var err error

	sqlstring := `SELECT intProjectRoleId, dtmCreated, dtmModified, intVersion, inbMserviceId, chvRoleName, chvDescription
	FROM tb_ProjectRoleType WHERE inbMserviceId = ? AND bitIsDeleted = 0`

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
		var role pb.ProjectRoleType

		err := rows.Scan(&role.ProjectRoleId, &created, &modified, &role.Version, &role.MserviceId, &role.RoleName, &role.Description)

		if err != nil {
			level.Error(s.logger).Log("what", "Scan", "error", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		role.Created = dml.DateTimeFromString(created)
		role.Modified = dml.DateTimeFromString(modified)

		resp.RoleTypes = append(resp.RoleTypes, &role)
	}

	return resp, err
}

// get current server version and uptime - health check
func (s *projService) GetServerVersion(ctx context.Context, req *pb.GetServerVersionRequest) (*pb.GetServerVersionResponse, error) {
	resp := &pb.GetServerVersionResponse{}

	currentSecs := time.Now().Unix()
	resp.ServerVersion = "v0.9.2"
	resp.ServerUptime = currentSecs - s.startSecs

	return resp, nil
}
