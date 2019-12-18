use mproject;

DROP TABLE IF EXISTS tb_StatusType;

-- MService project status type
CREATE TABLE tb_StatusType
(

    -- status identifier
    intStatusId INT NOT NULL,
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
    -- status name
    chvStatusName VARCHAR(32) NOT NULL,
    -- entity description
    chvDescription VARCHAR(255) NOT NULL,


    PRIMARY KEY (inbMserviceId,intStatusId),
    UNIQUE (inbMserviceId,chvStatusName)
) ENGINE=InnoDB;

