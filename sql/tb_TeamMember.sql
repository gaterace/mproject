use mproject;

DROP TABLE IF EXISTS tb_TeamMember;

-- MService project team member
CREATE TABLE tb_TeamMember
(

    -- team member id
    inbMemberId BIGINT AUTO_INCREMENT NOT NULL,
    -- creation date
    dtmCreated DATETIME NOT NULL,
    -- modification date
    dtmModified DATETIME NOT NULL,
    -- deletion date
    dtmDeleted DATETIME NOT NULL,
    -- has record been deleted?
    bitIsDeleted BOOL NOT NULL,
    -- version of this record
    intVersion INT NOT NULL,
    -- mservice account id
    inbMserviceId BIGINT NOT NULL,
    -- project identifier
    inbProjectId BIGINT NOT NULL,
    -- entity name
    chvName VARCHAR(32) NOT NULL,
    -- role id of this team member
    intProjectRoleId INT NOT NULL,
    -- email address of team member
    chvEmail VARCHAR(255) NOT NULL,


    PRIMARY KEY (inbMemberId),
    UNIQUE (inbProjectId,chvName)
) ENGINE=InnoDB;

