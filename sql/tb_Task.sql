use mproject;

DROP TABLE IF EXISTS tb_Task;

-- MService project task
CREATE TABLE tb_Task
(

    -- task identifier
    inbTaskId BIGINT AUTO_INCREMENT NOT NULL,
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
    -- entity description
    chvDescription VARCHAR(255) NOT NULL,
    -- status identifier
    intStatusId INT NOT NULL,
    -- project start date
    dtmStartDate DATETIME NOT NULL,
    -- project end date
    dtmEndDate DATETIME NOT NULL,
    -- task priority, 0 low to 9 high
    intPriority INT NOT NULL,
    -- parent task id
    inbParentId BIGINT NOT NULL,
    -- sibling position
    intPosition INT NOT NULL,


    PRIMARY KEY (inbTaskId),
    UNIQUE (inbProjectId,chvName)
) ENGINE=InnoDB;

