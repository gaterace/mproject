use mproject;

DROP TABLE IF EXISTS tb_TaskToMember;

-- MService map team member to task
CREATE TABLE tb_TaskToMember
(

    -- project identifier
    inbProjectId BIGINT NOT NULL,
    -- task identifier
    inbTaskId BIGINT NOT NULL,
    -- team member id
    inbMemberId BIGINT NOT NULL,
    -- creation date
    dtmCreated DATETIME NOT NULL,
    -- modification date
    dtmModified DATETIME NOT NULL,
    -- deletion date
    dtmDeleted DATETIME NOT NULL,
    -- has record been deleted?
    bitIsDeleted BOOL NOT NULL,
    -- mservice account id
    inbMserviceId BIGINT NOT NULL,
    -- hours allocated to task by team member
    decTaskHours DECIMAL(19,2) NOT NULL,


    PRIMARY KEY (inbProjectId,inbTaskId,inbMemberId)
) ENGINE=InnoDB;

