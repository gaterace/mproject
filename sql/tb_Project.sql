use mproject;

DROP TABLE IF EXISTS tb_Project;

-- MService project entity
CREATE TABLE tb_Project
(

    -- project identifier
    inbProjectId BIGINT AUTO_INCREMENT NOT NULL,
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
    -- entity name
    chvName VARCHAR(32) NOT NULL,
    -- entity description
    chvDescription VARCHAR(255) NOT NULL,
    -- status identifier
    intStatusId INT NOT NULL,
    -- project start date
    dtmStartDate DATETIME NOT NULL,
    -- project end date
    dtmEndDate DATETIME NOT NULL,


    PRIMARY KEY (inbProjectId),
    UNIQUE (inbMserviceId,chvName)
) ENGINE=InnoDB;

