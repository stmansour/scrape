--  FAA personnel database

DROP DATABASE IF EXISTS faa;
CREATE DATABASE faa;
USE faa;
GRANT ALL PRIVILEGES ON faa TO 'ec2-user'@'localhost';
GRANT ALL PRIVILEGES ON faa.* TO 'ec2-user'@'localhost';


-- **************************************
-- ****                              ****
-- ****         FAA DIRECTORY        ****
-- ****                              ****
-- **************************************

CREATE TABLE people (
    FID BIGINT NOT NULL AUTO_INCREMENT,
    FirstName VARCHAR(100) DEFAULT '',
    LastName VARCHAR(100) DEFAULT '',
    MiddleName VARCHAR(100) DEFAULT '',
    JobTitle VARCHAR(100) DEFAULT '',
    OfficePhone VARCHAR(100) DEFAULT '',
    OfficeFax VARCHAR(100) DEFAULT '',
    Email1 VARCHAR(50) DEFAULT '',
    MailAddress VARCHAR(50) DEFAULT '',
    MailAddress2 VARCHAR(50) DEFAULT '',
    MailCity VARCHAR(100) DEFAULT '',
    MailState VARCHAR(50) DEFAULT '',
    MailPostalCode VARCHAR(50) DEFAULT '',
    MailCountry VARCHAR(50) DEFAULT '',
    RoomNumber VARCHAR(50) DEFAULT '',
    MailStop VARCHAR(100) DEFAULT '',
    PRIMARY KEY (FID)     
);      
