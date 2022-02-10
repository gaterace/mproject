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
	"database/sql"

	"github.com/go-kit/kit/log/level"

	"github.com/gaterace/dml-go/pkg/dml"

	_ "github.com/go-sql-driver/mysql"

	pb "github.com/gaterace/mproject/pkg/mserviceproject"
)

// Generic response to set specific API method response.
type genericResponse struct {
	ErrorCode    int32
	ErrorMessage string
}

// Helper to build Project from projectId and mserviceId.
func (s *projService) GetProjectByIdHelper(projectId int64, mserviceId int64) (*genericResponse, *pb.Project) {
	resp := &genericResponse{}

	sqlstring := `SELECT p.inbProjectId, p.dtmCreated, p.dtmModified, p.intVersion, p.inbMserviceId, p.chvName, 
	p.chvDescription, p.intStatusId, p.dtmStartDate, p.dtmEndDate, s.chvStatusName 
	FROM tb_Project AS p 
    JOIN tb_StatusType AS s ON p.intStatusId = s.intStatusId
	WHERE p.inbProjectId = ? AND p.inbMserviceId = ? 
	AND p.bitIsDeleted = 0`

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

	var project pb.Project

	err = stmt.QueryRow(projectId, mserviceId).Scan(&project.ProjectId, &created, &modified, &project.Version,
		&project.MserviceId, &project.Name, &project.Description, &project.StatusId, &start_date, &end_date, &project.StatusName)

	if err == nil {
		project.Created = dml.DateTimeFromString(created)
		project.Modified = dml.DateTimeFromString(modified)
		project.StartDate = dml.DateTimeFromString(start_date)
		project.EndDate = dml.DateTimeFromString(end_date)
		resp.ErrorCode = 0
	} else if err == sql.ErrNoRows {
		resp.ErrorCode = 404
		resp.ErrorMessage = "not found"

	} else {
		level.Error(s.logger).Log("what", "QueryRow", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()

	}

	return resp, &project
}

// Helper to build Project from projectName and mserviceId.
func (s *projService) GetProjectByNameHelper(projectName string, mserviceId int64) (*genericResponse, *pb.Project) {
	resp := &genericResponse{}

	sqlstring := `SELECT p.inbProjectId, p.dtmCreated, p.dtmModified, p.intVersion, p.inbMserviceId, p.chvName, 
	p.chvDescription, p.intStatusId, p.dtmStartDate, p.dtmEndDate, s.chvStatusName 
	FROM tb_Project AS p 
    JOIN tb_StatusType AS s ON p.intStatusId = s.intStatusId
	WHERE p.chvName = ? AND p.inbMserviceId = ? 
	AND p.bitIsDeleted = 0`

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

	var project pb.Project

	err = stmt.QueryRow(projectName, mserviceId).Scan(&project.ProjectId, &created, &modified, &project.Version,
		&project.MserviceId, &project.Name, &project.Description, &project.StatusId, &start_date, &end_date, &project.StatusName)

	if err == nil {
		project.Created = dml.DateTimeFromString(created)
		project.Modified = dml.DateTimeFromString(modified)
		project.StartDate = dml.DateTimeFromString(start_date)
		project.EndDate = dml.DateTimeFromString(end_date)
		resp.ErrorCode = 0
	} else if err == sql.ErrNoRows {
		resp.ErrorCode = 404
		resp.ErrorMessage = "not found"

	} else {
		level.Error(s.logger).Log("what", "QueryRow", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = err.Error()

	}

	return resp, &project
}

// Helper to get the list of team members for a project.
func (s *projService) GetTeamMembersHelper(projectId int64, mserviceId int64) (*genericResponse, []*pb.TeamMember) {
	members := make([]*pb.TeamMember, 0)
	resp := &genericResponse{}

	sqlstring := `SELECT m.inbMemberId, m.dtmCreated, m.dtmModified, m.intVersion, m.inbMserviceId, m.inbProjectId, m.chvName,
	m.intProjectRoleId, m.chvEmail, r.chvRoleName
	FROM tb_TeamMember AS m
	JOIN tb_ProjectRoleType AS r ON m.intProjectRoleId = r.intProjectRoleId
	WHERE m.inbProjectId = ? AND m.inbMserviceId = ? AND m.bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(projectId, mserviceId)
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
		var member pb.TeamMember

		err := rows.Scan(&member.MemberId, &created, &modified, &member.Version, &member.MserviceId, &member.ProjectId,
			&member.Name, &member.ProjectRoleId, &member.Email, &member.RoleName)

		if err != nil {
			level.Error(s.logger).Log("what", "Scan", "error", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		member.Created = dml.DateTimeFromString(created)
		member.Modified = dml.DateTimeFromString(modified)
		members = append(members, &member)
	}

	return resp, members
}

// Helper to get the list of tasks for a project.
func (s *projService) GetProjectTasksHelper(projectId int64, mserviceId int64) (*genericResponse, []*pb.Task) {
	tasks := make([]*pb.Task, 0)
	resp := &genericResponse{}

	sqlstring := `SELECT t.inbTaskId, t.dtmCreated, t.dtmModified, t.intVersion, t.inbMserviceId, t.inbProjectId, t.chvName,
	t.chvDescription, t.intStatusId, t.dtmStartDate, t.dtmEndDate, t.intPriority, t.inbParentId, t.intPosition, s.chvStatusName 
	FROM tb_Task AS t 
	JOIN tb_StatusType AS s ON t.intStatusId = s.intStatusId 
	WHERE t.inbProjectId = ? AND t.inbMserviceId = ? AND t.bitIsDeleted = 0
	ORDER by t.inbParentId, t.intPosition`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(projectId, mserviceId)
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
		var start_date string
		var end_date string

		var task pb.Task

		err = rows.Scan(&task.TaskId, &created, &modified, &task.Version, &task.MserviceId,
			&task.ProjectId, &task.Name, &task.Description, &task.StatusId, &start_date, &end_date, &task.Priority, &task.ParentId,
			&task.Position, &task.StatusName)

		if err != nil {
			level.Error(s.logger).Log("what", "Scan", "error", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		task.Created = dml.DateTimeFromString(created)
		task.Modified = dml.DateTimeFromString(modified)
		task.StartDate = dml.DateTimeFromString(start_date)
		task.EndDate = dml.DateTimeFromString(end_date)

		tasks = append(tasks, &task)
	}

	return resp, tasks
}

// Helper to get the list of TaskWrapper objects for a project.
func (s *projService) GetProjectTaskWrapperHelper(projectId int64, mserviceId int64) (*genericResponse, []*pb.TaskWrapper) {
	wraps := make([]*pb.TaskWrapper, 0)

	resp, tasks := s.GetProjectTasksHelper(projectId, mserviceId)

	if resp.ErrorCode != 0 {
		return resp, nil
	}

	wtaskMap := make(map[int64]*pb.TaskWrapper)
	for _, task := range tasks {
		wrap := convertTaskToWrapper(task)
		wraps = append(wraps, wrap)
		wtaskMap[wrap.GetTaskId()] = wrap
	}

	for _, wrap := range wraps {
		if wrap.GetParentId() != 0 {
			parentTask, ok := wtaskMap[wrap.GetParentId()]
			if ok {
				parentTask.ChildTaskWrappers = append(parentTask.ChildTaskWrappers, wrap)
			}
		}
	}

	memberMap := make(map[int64]*pb.TeamMember)

	resp, members := s.GetTeamMembersHelper(projectId, mserviceId)

	if resp.ErrorCode != 0 {
		return resp, nil
	}
	for _, member := range members {
		memberMap[member.GetMemberId()] = member
	}

	resp, mbrtasks := s.GetTaskToMemberHelper(projectId, mserviceId)
	if resp.ErrorCode != 0 {
		return resp, nil
	}

	for _, mbrtask := range mbrtasks {
		wtask, ok := wtaskMap[mbrtask.GetTaskId()]
		if ok {
			mbr, ok := memberMap[mbrtask.GetMemberId()]
			if ok {
				member := *mbr
				member.TaskHours = mbrtask.GetTaskHours()
				wtask.TeamMembers = append(wtask.TeamMembers, &member)
			}
		}
	}

	return resp, wraps
}

// Helper to get the list of TaskToMember mappings fpr a project.
func (s *projService) GetTaskToMemberHelper(projectId int64, mserviceId int64) (*genericResponse, []*pb.TaskToMember) {
	mbrtasks := make([]*pb.TaskToMember, 0)
	resp := &genericResponse{}

	sqlstring := `SELECT inbProjectId, inbTaskId, inbMemberId, dtmCreated, dtmModified, 
	inbMserviceId, decTaskHours FROM tb_TaskToMember WHERE inbProjectId = ? AND inbMemberId = ? 
	AND bitIsDeleted = 0`

	stmt, err := s.db.Prepare(sqlstring)
	if err != nil {
		level.Error(s.logger).Log("what", "Prepare", "error", err)
		resp.ErrorCode = 500
		resp.ErrorMessage = "db.Prepare failed"
		return resp, nil
	}

	defer stmt.Close()

	rows, err := stmt.Query(projectId, mserviceId)
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
		var t2m pb.TaskToMember

		err := rows.Scan(&t2m.ProjectId, &t2m.TaskId, &t2m.MemberId, &created, &modified, &t2m.MserviceId, &task_hours)
		if err != nil {
			level.Error(s.logger).Log("what", "Scan", "error", err)
			resp.ErrorCode = 500
			resp.ErrorMessage = err.Error()
			return resp, nil
		}

		t2m.Created = dml.DateTimeFromString(created)
		t2m.Modified = dml.DateTimeFromString(modified)
		d, err := dml.DecimalFromString(task_hours)
		if err == nil {
			t2m.TaskHours = d
		}

		mbrtasks = append(mbrtasks, &t2m)
	}

	return resp, mbrtasks
}

// Helper to convert a Project to a ProjectWrapper.
func convertProjectToWrapper(project *pb.Project) *pb.ProjectWrapper {
	wrap := pb.ProjectWrapper{}
	wrap.ProjectId = project.GetProjectId()
	wrap.Created = project.GetCreated()
	wrap.Modified = project.GetModified()
	wrap.Deleted = project.GetDeleted()
	wrap.Version = project.GetVersion()
	wrap.MserviceId = project.GetMserviceId()
	wrap.Name = project.GetName()
	wrap.Description = project.GetDescription()
	wrap.StatusId = project.GetStatusId()
	wrap.StartDate = project.GetStartDate()
	wrap.EndDate = project.GetEndDate()
	wrap.StatusName = project.GetStatusName()

	return &wrap
}

// Helper to convert a Task to a TaskWrapper.
func convertTaskToWrapper(task *pb.Task) *pb.TaskWrapper {
	wrap := pb.TaskWrapper{}
	wrap.TaskId = task.GetTaskId()
	wrap.Created = task.GetCreated()
	wrap.Modified = task.GetModified()
	wrap.Deleted = task.GetDeleted()
	wrap.Version = task.GetVersion()
	wrap.MserviceId = task.GetMserviceId()
	wrap.ProjectId = task.GetProjectId()
	wrap.Name = task.GetName()
	wrap.Description = task.GetDescription()
	wrap.StatusId = task.GetStatusId()
	wrap.StatusName = task.GetStatusName()
	wrap.StartDate = task.GetStartDate()
	wrap.EndDate = task.GetEndDate()
	wrap.Priority = task.GetPriority()
	wrap.ParentId = task.GetParentId()
	wrap.Position = task.GetPosition()

	return &wrap
}
