use mproject;

DROP TABLE IF EXISTS tb_ProjectRoleType;

-- MService project role type
CREATE TABLE tb_ProjectRoleType
(

    -- role id of this team member
    intProjectRoleId INT NOT NULL,
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
    -- role name of this team member
    chvRoleName VARCHAR(32) NOT NULL,
    -- entity description
    chvDescription VARCHAR(255) NOT NULL,


    PRIMARY KEY (inbMserviceId,intProjectRoleId),
    UNIQUE (inbMserviceId,chvRoleName)
) ENGINE=InnoDB;

