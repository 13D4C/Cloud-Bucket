/*M!999999\- enable the sandbox mode */
-- MariaDB dump 10.19  Distrib 10.5.29-MariaDB, for Linux (x86_64)
--
-- Host: terraform-20251031181255777500000007.cvv9ubjqzzyf.us-east-1.rds.amazonaws.com    Database: Clown_Project_v1
-- ------------------------------------------------------
-- Server version       10.11.14-MariaDB

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `DEFINE_ROLE`
--

DROP TABLE IF EXISTS `DEFINE_ROLE`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `DEFINE_ROLE` (
  `NO_ROLE` int(11) DEFAULT NULL,
  `ADMIN` int(11) DEFAULT NULL,
  `STUDENT` int(11) DEFAULT NULL,
  `PROFESSOR` int(11) DEFAULT NULL,
  `OFFICER` int(11) DEFAULT NULL
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `DEFINE_ROLE`
--

LOCK TABLES `DEFINE_ROLE` WRITE;
/*!40000 ALTER TABLE `DEFINE_ROLE` DISABLE KEYS */;
/*!40000 ALTER TABLE `DEFINE_ROLE` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `FILE_LIST`
--

DROP TABLE IF EXISTS `FILE_LIST`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `FILE_LIST` (
  `FILE_ID` int(11) NOT NULL AUTO_INCREMENT,
  `OWNER_ID` int(11) DEFAULT NULL,
  `FILE_NAME` varchar(100) DEFAULT NULL,
  `FILE_TYPE` varchar(100) DEFAULT NULL,
  `FILE_SIZE` bigint(20) DEFAULT NULL,
  `FILE_PATH` text DEFAULT NULL,
  `STATUS` varchar(100) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `modified_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`FILE_ID`),
  KEY `FILE_LIST_USERS_FK` (`OWNER_ID`),
  CONSTRAINT `FILE_LIST_USERS_FK` FOREIGN KEY (`OWNER_ID`) REFERENCES `USERS` (`USER_ID`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=376 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `FILE_LIST`
--

LOCK TABLES `FILE_LIST` WRITE;
/*!40000 ALTER TABLE `FILE_LIST` DISABLE KEYS */;
INSERT INTO `FILE_LIST` VALUES (319,19,'data.json','application/json',333,'/test/15-060_classroom-1','active','2025-09-20 15:40:57','2025-09-20 15:40:57'),(320,19,'index.html','text/html',66100,'/test/15-060_classroom-1','active','2025-09-20 15:40:57','2025-09-20 15:40:57'),(321,19,'+page.svelte','',8583,'/test/15-060_classroom-1','active','2025-09-20 17:30:10','2025-09-20 17:30:10'),(322,19,'+page.svelte','',36140,'/test/15-060_classroom-1','active','2025-09-20 20:02:01','2025-09-20 20:02:01'),(323,22,'Huawei_Talent_Document-HighQuality (1).pdf','application/pdf',6712532,'/','active','2025-09-24 15:07:10','2025-09-24 15:07:10'),(324,22,'kung.pdf','application/pdf',24457087,'/','active','2025-09-24 15:07:24','2025-09-24 15:07:24'),(325,22,'kungpen.pdf','application/pdf',61617931,'/','active','2025-09-24 15:07:24','2025-09-24 15:07:24'),(335,19,'data.json','application/json',343,'/mit15_060f14_hw3_exec-1','active','2025-09-26 18:21:42','2025-09-26 18:21:42'),(339,19,'FGT_VM64_KVM-v7.0.9.M-build0444-FORTINET.out.kvm','',0,'/','active','2025-09-27 13:32:41','2025-09-27 13:32:41'),(340,19,'PRNAS','',0,'/','active','2025-09-27 13:33:09','2025-09-27 13:33:09'),(342,19,'Paper.png','image/png',7824635,'/PRNAS','active','2025-09-27 13:33:28','2025-09-27 13:33:28'),(343,23,'Screenshot From 2025-09-10 16-05-35.png','image/png',222159,'/','active','2025-10-02 16:59:33','2025-10-02 16:59:33'),(346,19,'21-ปกหลัง.pdf','application/pdf',128846,'/01-CCH','active','2025-10-13 14:59:33','2025-10-13 14:59:33'),(348,19,'02-Final_Content.pdf','application/pdf',101991,'/01-CCH','active','2025-10-13 14:59:34','2025-10-13 14:59:34'),(352,19,'11-Final_Chapter8.pdf','application/pdf',677074,'/01-CCH','active','2025-10-13 14:59:34','2025-10-13 14:59:34'),(353,19,'09-Final_Chapter6.pdf','application/pdf',254080,'/01-CCH','active','2025-10-13 14:59:34','2025-10-13 14:59:34'),(357,19,'05-Final_Chapter2.pdf','application/pdf',2663992,'/01-CCH','active','2025-10-13 14:59:34','2025-10-13 14:59:34'),(358,19,'16-Appendix_d.pdf','application/pdf',133782,'/01-CCH','active','2025-10-13 14:59:34','2025-10-13 14:59:34'),(361,19,'14-Appendix_b.pdf','application/pdf',115291,'/01-CCH','active','2025-10-13 14:59:34','2025-10-13 14:59:34'),(364,19,'17-Appendix_e.pdf','application/pdf',105517,'/01-CCH','active','2025-10-13 14:59:34','2025-10-13 14:59:34'),(366,19,'20-ใบคั่น.pdf','application/pdf',110898,'/01-CCH','active','2025-10-13 14:59:34','2025-10-13 14:59:34'),(369,23,'4f422a05-678e-4c07-80ac-1154aa4787b5.pdf','application/pdf',20249,'/','active','2025-10-31 19:44:33','2025-10-31 19:44:33'),(370,23,'dump-Clown_Project_v1-202511010155.sql','',14702,'/','active','2025-10-31 20:16:57','2025-10-31 20:16:57'),(371,23,'4f422a05-678e-4c07-80ac-1154aa4787b5.pdf','application/pdf',20249,'/','active','2025-10-31 20:17:00','2025-10-31 20:17:00'),(372,23,'66a47769-7e62-44c4-8ad8-609c03dd4b25.pdf','application/pdf',19340,'/','active','2025-10-31 20:17:02','2025-10-31 20:17:02'),(373,23,'iot COAP (1).mp4','video/mp4',1959022,'/','active','2025-10-31 20:17:09','2025-10-31 20:17:09'),(374,23,'VID_20251016_160532_8K.mp4','video/mp4',304099874,'/','active','2025-10-31 20:19:46','2025-10-31 20:19:46'),(375,23,'DEVASC-disk1.vmdk','application/x-virtualbox-vmdk',24245829632,'/','active','2025-10-31 21:30:04','2025-10-31 21:30:04');
/*!40000 ALTER TABLE `FILE_LIST` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `FOLDER_LIST`
--

DROP TABLE IF EXISTS `FOLDER_LIST`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `FOLDER_LIST` (
  `FOLDER_ID` varchar(100) NOT NULL,
  `OWNER_ID` int(11) NOT NULL,
  `FOLDER_NAME` text NOT NULL,
  `PATH` text NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `modified_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `STATUS` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`FOLDER_ID`),
  KEY `FOLDER_LIST_USERS_FK` (`OWNER_ID`),
  CONSTRAINT `FOLDER_LIST_USERS_FK` FOREIGN KEY (`OWNER_ID`) REFERENCES `USERS` (`USER_ID`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `FOLDER_LIST`
--

LOCK TABLES `FOLDER_LIST` WRITE;
/*!40000 ALTER TABLE `FOLDER_LIST` DISABLE KEYS */;
INSERT INTO `FOLDER_LIST` VALUES ('09b6f774-c410-4f24-b726-1ee27df827cb',24,'eiei','/','2025-10-13 14:55:00','2025-10-13 14:55:00','active'),('75b9ff4a-2150-4185-be20-df99b6df273b',19,'PRNAS','/','2025-09-27 13:33:28','2025-09-27 13:33:28','active'),('8153a726-973e-49bc-9d63-017ab14476a5',19,'mit15_060f14_hw3_exec-1','/','2025-09-26 18:21:41','2025-09-26 18:21:41','active'),('b1185863-c1f8-4961-a630-73b00e4ab09e',19,'15-060_classroom-1','/test','2025-09-20 15:40:57','2025-09-20 15:40:57','active'),('b94acc8c-5a86-430f-ad81-51a502de4322',19,'test','/','2025-09-20 15:40:52','2025-09-20 15:40:52','active'),('d992d24a-c013-4888-9b12-15ba8e0c747d',19,'01-CCH','/','2025-10-13 14:59:33','2025-10-13 14:59:33','active'),('e15c1dce-a6a4-4d05-9d5d-18fb21ad0add',22,'d','/','2025-10-13 15:01:15','2025-10-13 15:01:15','active');
/*!40000 ALTER TABLE `FOLDER_LIST` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `GROUP_LIST`
--

DROP TABLE IF EXISTS `GROUP_LIST`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `GROUP_LIST` (
  `GROUP_ID` int(11) NOT NULL AUTO_INCREMENT,
  `GROUP_OWNER_ID` int(11) DEFAULT NULL,
  `GROUP_NAME` varchar(100) DEFAULT NULL,
  `GROUP_DEFAULT_QUOTA` int(11) DEFAULT NULL,
  `GROUP_PRIVILLAGE` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`GROUP_ID`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `GROUP_LIST`
--

LOCK TABLES `GROUP_LIST` WRITE;
/*!40000 ALTER TABLE `GROUP_LIST` DISABLE KEYS */;
/*!40000 ALTER TABLE `GROUP_LIST` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `GROUP_MANAGABLE`
--

DROP TABLE IF EXISTS `GROUP_MANAGABLE`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `GROUP_MANAGABLE` (
  `ADMIN_GROUP_ID` int(11) DEFAULT NULL,
  `USER_GROUP_ID` int(11) DEFAULT NULL
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `GROUP_MANAGABLE`
--

LOCK TABLES `GROUP_MANAGABLE` WRITE;
/*!40000 ALTER TABLE `GROUP_MANAGABLE` DISABLE KEYS */;
/*!40000 ALTER TABLE `GROUP_MANAGABLE` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `GROUP_MEMBERS`
--

DROP TABLE IF EXISTS `GROUP_MEMBERS`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `GROUP_MEMBERS` (
  `GROUP_ID` int(11) DEFAULT NULL,
  `USER_ID` int(11) DEFAULT NULL
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `GROUP_MEMBERS`
--

LOCK TABLES `GROUP_MEMBERS` WRITE;
/*!40000 ALTER TABLE `GROUP_MEMBERS` DISABLE KEYS */;
/*!40000 ALTER TABLE `GROUP_MEMBERS` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `SHARED_FILE`
--

DROP TABLE IF EXISTS `SHARED_FILE`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `SHARED_FILE` (
  `USER_ID` int(11) NOT NULL,
  `FILE_ID` int(11) NOT NULL,
  `PERMISSION` varchar(100) NOT NULL,
  KEY `SHARED_FILE_USERS_FK` (`USER_ID`),
  KEY `SHARED_FILE_FILE_LIST_FK` (`FILE_ID`),
  CONSTRAINT `SHARED_FILE_FILE_LIST_FK` FOREIGN KEY (`FILE_ID`) REFERENCES `FILE_LIST` (`FILE_ID`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `SHARED_FILE_USERS_FK` FOREIGN KEY (`USER_ID`) REFERENCES `USERS` (`USER_ID`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `SHARED_FILE`
--

LOCK TABLES `SHARED_FILE` WRITE;
/*!40000 ALTER TABLE `SHARED_FILE` DISABLE KEYS */;
INSERT INTO `SHARED_FILE` VALUES (21,319,'write'),(21,320,'write'),(21,321,'write'),(22,342,'read'),(22,346,'read'),(22,348,'read'),(22,352,'read'),(22,353,'read'),(22,357,'read'),(22,358,'read'),(22,361,'read'),(22,364,'read'),(22,366,'read');
/*!40000 ALTER TABLE `SHARED_FILE` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `SHARED_FOLDER`
--

DROP TABLE IF EXISTS `SHARED_FOLDER`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `SHARED_FOLDER` (
  `USER_ID` int(11) NOT NULL,
  `FOLDER_ID` varchar(100) NOT NULL,
  `PERMISSION` varchar(100) NOT NULL,
  KEY `SHARED_FOLDER_FOLDER_LIST_FK` (`FOLDER_ID`),
  KEY `SHARED_FOLDER_USERS_FK` (`USER_ID`),
  CONSTRAINT `SHARED_FOLDER_FOLDER_LIST_FK` FOREIGN KEY (`FOLDER_ID`) REFERENCES `FOLDER_LIST` (`FOLDER_ID`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `SHARED_FOLDER_USERS_FK` FOREIGN KEY (`USER_ID`) REFERENCES `USERS` (`USER_ID`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `SHARED_FOLDER`
--

LOCK TABLES `SHARED_FOLDER` WRITE;
/*!40000 ALTER TABLE `SHARED_FOLDER` DISABLE KEYS */;
INSERT INTO `SHARED_FOLDER` VALUES (21,'b94acc8c-5a86-430f-ad81-51a502de4322','write'),(21,'b1185863-c1f8-4961-a630-73b00e4ab09e','write'),(22,'75b9ff4a-2150-4185-be20-df99b6df273b','read'),(22,'d992d24a-c013-4888-9b12-15ba8e0c747d','read');
/*!40000 ALTER TABLE `SHARED_FOLDER` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `USERS`
--

DROP TABLE IF EXISTS `USERS`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `USERS` (
  `USER_ID` int(11) NOT NULL AUTO_INCREMENT,
  `USERNAME` varchar(100) DEFAULT NULL,
  `PASSWORD` varchar(100) DEFAULT NULL,
  `DISPLAYNAME` varchar(100) DEFAULT NULL,
  `EMAIL` varchar(100) DEFAULT NULL,
  `PHONE` varchar(100) DEFAULT NULL,
  `BIRTHDATE` date DEFAULT NULL,
  `ADDRESS` varchar(100) DEFAULT NULL,
  `ROLE` varchar(100) DEFAULT NULL,
  `USER_QUOTA` bigint(20) NOT NULL DEFAULT 0,
  `USED_QUOTA` bigint(20) NOT NULL DEFAULT 0,
  `STATUS` varchar(255) DEFAULT '',
  PRIMARY KEY (`USER_ID`)
) ENGINE=InnoDB AUTO_INCREMENT=27 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `USERS`
--

LOCK TABLES `USERS` WRITE;
/*!40000 ALTER TABLE `USERS` DISABLE KEYS */;
INSERT INTO `USERS` VALUES (10,'e','$2a$10$D5Gu5QxEDx/lVNSqQeRdTujLUMpcqoi9imxBsr1mFjradcnjzMehy','-','-','-',NULL,'-','User',1000000000,0,'Active'),(12,'dgdrhdh','$2a$10$2FjZC7q6hraHSPLCd8iseuZlPvZqYSV8UcHqLtQV4pBBkYNq3sG7m','-','-','-',NULL,'-','User',100000000,0,'Active'),(13,'pan','$2a$10$7G2BXSrHbHHQ/C/7I2qmC.tLy1plVgUnqSNjGw4Xnc6gcaDyG69kG','-','-','-',NULL,'-','User',100000000,0,'Active'),(18,'r','$2a$10$jsg/7f7J1kyCM5dS511j1OqtBVxECN5idCd7oyVdwQDOJH0Akb3dy','-','r@gmail.com','1236984567',NULL,'-','User',100000000,0,'Active'),(19,'Anner','$2a$10$V7XbTKt/h063AhLYVuJTo.HREYIQn7HF2ZZgZqOd/JRQokiPDGdqy','-','icetherockth@gmail.com','0931980805',NULL,'-','Admin',500000000,12116116,'Active'),(20,'asfg','$2a$10$NNsMkg1I1p7VnYJel7a7e.gmPp6txhn160jJsSKrRNF.AgoKYI6s6',NULL,'asfdg@gmail.com','1234567890',NULL,NULL,'User',0,0,'Active'),(21,'nigga','$2a$10$NUDddZxEVv2Q8NavvWnkyeCJDn6/Gq9bHJJGAcrblu38uZyzqHimG',NULL,'Nigga@gmail.com','0123456789',NULL,NULL,'User',0,0,'Active'),(22,'d','$2a$10$U5B30zgzyHMf732c1dEkRuYBu/noOSVgA6M8q2YL.Gm.t06mcSL92',NULL,'66070216@kmitl.ac.th','0852345893',NULL,NULL,'User',0,0,'Active'),(23,'khaow','$2a$10$vQ1Mk86Bm2n/LaxQDEjz7OtglW25EEJn2kmo9ygkYtu/kcg/BJ332',NULL,'khaow1000IQ@dixktator.CCCP','911',NULL,NULL,'Admin',9000000000000,24552185227,'Active'),(24,'cloud','$2a$10$VgocG/oWCexZaItA6uKOf.EEpn.xlJjVmHw5H2exb1fnkJFtnURVm',NULL,'cloud@gmail.com','0854263698',NULL,NULL,'User',0,0,''),(25,'test1','$2a$10$O1qQZ5OqbRltG059Rq/xAOziGTDHGGw3.FMsAG9OqORHBimqumn6C',NULL,'test1@gmail.com','12345',NULL,NULL,'User',0,0,''),(26,'test2','$2a$10$NbLVTLNAVWJ5qYGeO9AE7uRQM47wF957c0LiOVLt..HeZSnjDKUz.',NULL,'test2@G.c','22222',NULL,NULL,'User',100000,0,'Active');
/*!40000 ALTER TABLE `USERS` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2025-10-31 21:39:43
