<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
#backUpPane .el-form-item{
	padding:10px 0px 10px 0px;
	border:1px solid #DEDFE6;
	width:59%;
	margin-left:40px;
	height:41px;
	box-sizing:border-box;
}
#backUpPane  .el-form-item__content{
	margin-left:19px !important;
}
#backUpPane  .labelTitle{
	white-space:nowrap;
	font-size:14px;
	color:#5A7b92;
	font-weight:400;
	height:19px;
	line-height:19px;
	margin-left:40px;
	margin-bottom:11px;
}
#backUpPane  .nomargin-input .el-form-item__content{
	margin-left:0px !important;
}
#backUpPane  .el-tabs__item{
	font-weight:700;
	font-style:normal;
	font-size:14px;
}
#backUpPane  .el-radio__label{
	color:#000000;
}
#backUpPane  .el-checkbox{
	color:#000000;
}
#backUpPane  .el-input__icon{
	line-height:25px;
}
#backUpPane .dayContainer .el-form-item__content{
	margin-left:0px !important;
}
#backUpPane .el-input.el-input--small{
	width:unset;
}
#backUpPane .el-button{
	padding:unset;
}
#backUpPane .list-item{
	display:flex;
	width:90%;
	height:60px;
	margin-left:40px;
	align-items:center;
	padding-bottom:20px;
	border-bottom:1px solid #DEDFE6;
}
#backUpPane .backupIcon{
	width:27px;
	height:24px;
	background:yellow;
}
#backUpPane .backupInfo{
	/* display:flex; */
	/* flex:2 1 auto; */
	/* height:40px;
	margin-left:25px;
	width:300px; */
	font-weight:400;
	color:#000000;
	
}
#backRecoverPage .el-tabs{
	height:100%;
}
#backUpPane .backupInfo > span{
	font-size:14px;
	display:inline-block;
	height:15px;
	line-height:20px;
	color:#000000;
	font-weight:400;
	margin-bottom:10px;
}
#backUpPane .backupTime{
	height:40px;
	margin-left:25px;
	width:150px;
}
#backUpPane .backupProgress{
	width:200px;
}
#backUpPane .el-table__header-wrapper{
	display:none
}
#backUpPane .el-table--border,#backUpPane .el-pagination{
	border:none;
}
#backUpPane .el-table td{
	padding:20px 0;
	border-right:none;
	padding:10px 0px;
}
#backUpPane .btnStyle{
	vertical-align:middle;
	text-align:center;
	display:inline-block;
	width:83px;
	height:21px;
	line-height:21px;
	cursor:pointer;
	margin-left:18px;
	border-radius:2px;
}
#backUpPane .disEnableBtnSty{
	background:#b0cbdd;
	color:#ffffff;
}
#backUpPane .el-checkbox-group{
	display:flex;
	flex-direction:column;
}
#backUpPane .el-checkbox+.el-checkbox{
	margin-left:0px;
}
#backUpPane .chekboxContainer{
	display:flex;
	align-items:center;
	margin-top:15px;
	
}
#backUpPane .chekboxContainer .el-checkbox{
	min-width:170px;
	
}
#backUpPane .chekboxContainer .el-input{
	width:180px;
	margin-left:20px;
	margin-right:15px;
}
#backUpPane .titleBox{
	width:47%;
	border:1px solid #91D5FF;
	background-color:#E6F6FF;
	height:37px;
	line-height:37px;
	margin-left:40px;
	border-radius:3px;
	padding-left:13px;
	margin-bottom:11px;
}
#backUpPane .chekboxContainer .el-form-item{
	padding:unset;
	border:unset;
	width:unset;
	margin-left:unset;
	height:unset;
	align-self:flex-end;
}
#backUpPane .chekboxContainer .el-form-item__error{
	left:23px;
	font-size:11px;
}

/* 开始---恢复 */
#backRecoverPage .addButtonDiv{
	position:absolute;
	right:20px;
	top:20px;
	width:36px;
}
#backRecoverPage .addButtonText{
	text-align:center;
	font-size:16px;
	color:#1DA3FC;
	margin-top:5px;
}
#backRecoverPage .tableDiv{
	height:26px;
}
.backRecoverDialogCls{
	background:none;
	box-shadow:none;
}
.backRecoverDialogCls .el-form-item__label{
	font-size:14px;
	color:#5A7B92;
}
.backRecoverDialogCls .el-dialog__title{
	font-weight:700;
	font-size:15px;
	color:#5A7B92;
}
.backRecoverDialogCls .el-dialog__headerbtn{
	top:15px;
}
.backRecoverDialogCls .el-card.is-always-shadow{
	width:463px;
	height:360px;
}
.backRecoverDialogCls .el-dialog__body{
	padding:0px;
}
.backRecoverDialogCls .el-card__footer{
	border:none;
}
.backRecoverDialogCls .backup_start,.recoverBg2,.recover_start{
	width:600px;
	height:360px;
	position:relative
}
.backRecoverDialogCls .backup_start span:nth-child(1){
	 font-size:28px;
	 color:#fff;
	 font-weight:bold;
	 position:absolute;
	 left:91px;
	 top:79px;
}
/* .backup_start span:nth-child(2){
	  font-size:14px;
	  color:#fff;
	  font-weight:300;
	  position:absolute;
	  left:91px;
	  top:123px;
} */
.backRecoverDialogCls .backup_start p:nth-child(2){
	   width:120px;
	   height:36px;
	   background:#fff;
	   border-radius:100px;
	   font-size:16px;
	   font-weight:400;
	   color:#1DA3FC;
	   line-height:36px;
	   text-align:center;
	   cursor:pointer;
	   position:absolute;
	   left:433px;
	   top:92px;
}
.backRecoverDialogCls .backup_start p:nth-child(3){
	   width:34px;
	   height:36px;
	   position:absolute;
	   left:208px;
	   top:273px;
}
.backRecoverDialogCls .backup_start p:nth-child(4){
	   font-size:16px;
	   font-weight:400;
	   color:#8BCEF3;
	   position:absolute;
	   left:264px;
	   top:280px;
}
.backRecoverDialogCls .recoverBg2 p:nth-child(1){
	font-size:28px;
	width:448px;
	font-weight:bold;
	color:#fff;
	width:600px;
	text-align:center;
	position:absolute;
	top:90px;
}
.backRecoverDialogCls .recoverBg2 p:nth-child(2){
	width:120px;
	height:36px;
	background:#fff;
	border-radius:100px;
	font-size:16px;
	font-weight:400;
	color:#1DA3FC;
	position:absolute;
	left:218px;
	top:229px;
	line-height:36px;
	text-align:center;
	cursor:pointer;
}
.backRecoverDialogCls .recoverBg2 p:nth-child(3){
	font-size:16px;
	font-weight:400;
	color:#fff;
	text-decoration:underline;
	position:absolute;
	left:350px;
	top:237px;
	cursor:pointer;
}
.backRecoverDialogCls .backupFail,.backupSuccess{
	width:600px;
	height:360px;
	background:#fff;
	position:relative;
	cursor:pointer;
}
.backRecoverDialogCls .backupFail p:nth-child(1){
	width:76px;
	height:76px;
	position:absolute;
	left:262px;
	top:51px;
}
.backRecoverDialogCls .backupFail p:nth-child(2){
	font-size:24px;
	color:#333;
	font-weight:400;
	width:600px;
	text-align:center;
	position:absolute;
	top:139px;
}
.backRecoverDialogCls .backupFail p:nth-child(3){
	width:336px;
	font-size:14px;
	color:#A2A2A2;
	font-weight:400;
	width:600px;
	text-align:center;
	position:absolute;
	top:190px;
}
.backRecoverDialogCls .backupFail p:nth-child(4){
	width:120px;
	height:36px;
	line-height:36px;
	text-align:center;
	background:#1DA3FC;
	color:#fff;
	font-size:16px;
	font-weight:400;
	position:absolute;
	left:240px;
	top:273px;
	border-radius:100px;
	cursor:pointer;
}
.backRecoverDialogCls .backupFail p:nth-child(5){
	width:120px;
	height:36px;
	line-height:36px;
	text-align:center;
	background:#1DA3FC;
	color:#fff;
	font-size:16px;
	font-weight:400;
	position:absolute;
	left:309px;
	top:273px;
	border-radius:100px;
	cursor:pointer;
}
.backRecoverDialogCls .recover_start p:nth-child(1){
	width:600px;
	text-align:center;
	font-size:28px;
	font-weight:bold;
	color:#fff;
	position:absolute;
	top:94px;
}
.backRecoverDialogCls .recover_start p:nth-child(2){
	width:34px;
	height:36px;
	position:absolute;
	left:189px;
	top:273px;
}
.backRecoverDialogCls .recover_start p:nth-child(3){
	font-size:16px;
	font-weight:400;
	color:#8BCEF3;
	position:absolute;
	left:236px;
	top:280px;
}
.el-input.el-input--small,.el-textarea.el-input--small{
	width:146px;
}
#backRecoverPage .searchCon{
	width:400px;
	margin-left:27px;
}
#backRecoverPage .searchCon .el-input{
	width:100%;
}
#backRecoverPage .searchCon .el-input__inner{
	border-radius:4px;
	background:#FFFFFF;
	border: 1px solid #E9E9E9;
	height:30px;
	line-height:30px;
}
#backRecoverPage .searchCon{
	width:400px;
	margin-left:27px;
}
#backRecoverPage .searchCon .el-input{
	width:100%;
}
#backRecoverPage .searchCon .el-input__inner{
	border-radius:4px;
	background:#FFFFFF;
	border: 1px solid #E9E9E9;
	height:30px;
	line-height:30px;
}
/* 结束---恢复 */
</style>
<div class="panelDefault" id="backRecoverPage">
	<template>
		<el-upload :on-success='importFile' :show-file-list=false action="${ctx}/sys/backupAndRecovery/importBackupFile.action">
        	<!-- <div v-if="recoverFlag" class="circleIcon placeholder-bt CODE_SYSTEM_BACKUP_RESTORE hidden" style="z-index:10;top:40px;right:20px;" placeholder="<%=rb.getString("DaoRu")%>"> -->
			<div v-if="false" class="circleIcon placeholder-bt CODE_SYSTEM_BACKUP_RESTORE hidden" style="z-index:10;top:40px;right:20px;" placeholder="<%=rb.getString("DaoRu")%>">
				<span class="el-icon el-icon-circle-import"></span>
			</div>
		</el-upload>
		<el-tabs v-model="activeName" @tab-click="handeClick">
			<el-tab-pane label="<%=rb.getString("BeiFen")%>" name="backup" id="backUpPane" style='flex: 1 1 100%; display: flex; flex-direction: column;height: 100%;'>
				<!-- form 表单校验 -->
                    <el-form :rules="rules" ref="backupform" :model="form" label-width="80px" style="margin-top:20px; flex: 1 1 0%; overflow: auto;" :disabled="formShow">
                          <div class="labelTitle"><%=rb.getString("ZhiXingFangShi")%></div>
                           <el-form-item label="">
                            <el-radio-group v-model="form.resource">
                              <el-radio label="1"><%=rb.getString("LiJiBeiFen")%></el-radio>
                              <el-radio label="2"><%=rb.getString("DingShiBeiFen")%></el-radio>
                            </el-radio-group>
                          </el-form-item>
                          <div class="dayContainer" style="display:flex;width:47%;margin-left:40px" v-if="form.resource == '2' "> <!-- v-if="form.resource == '1' " -->
                               <div style="flex:2 1 auto;">
                                    <div class="labelTitle" style="margin-left:0px;margin-bottom:0px;"><%=rb.getString("KaiShiShiJian")%></div>
                                    <el-form-item label="" style="border:none;margin-left:0px;" prop='date1'>
                                        <el-date-picker format="yyyy-MM-dd HH:mm:ss" value-format="yyyy-MM-dd HH:mm:ss" type="datetime" placeholder="" v-model="form.date1" style="width: 100%;"></el-date-picker>
                                    </el-form-item>
                               </div>
                          </div>

					  <div v-if="form.resource == '2'" class="labelTitle"><%=rb.getString("ZhouQi")%></div>
					  <el-form-item v-if="form.resource == '2'" label="" prop="period">
					   	<el-select v-model="form.period" placeholder="">
						  <el-option v-for="item in periodOptions" :key="item.value" :label="item.label" :value="item.value"></el-option>
						</el-select>
					  </el-form-item>
					  
					  <div class="labelTitle"><%=rb.getString("BeiFenNeiRong")%></div>
					  <div style="display:flex;align-items:flex-end;">
						<el-form-item label="" style="height:150px;">
							<el-checkbox-group v-model="form.type">
							<div class="chekboxContainer">
									<el-checkbox label="system_config" name="type"><%=rb.getString("XiTongPeiZhi")%></el-checkbox>
									<el-checkbox label="system_data" name="type"><%=rb.getString("XiTongShuJu")%></el-checkbox>
							</div>
							</el-checkbox-group>
						</el-form-item>
						<div v-if="form.resource == '1' " class="CODE_SYSTEM_BACKUP_RESTORE hidden">  <el-button type="primary" style="height:28px;width:80px;margin-bottom:16px;margin-left:200px" @click="startBackup(1)"><%=rb.getString("KaiShi")%></el-button></div>
					 </div>
					 <div style="display:flex;align-items:flex-end;"  v-if="form.resource == '2' ">
						 <div class="titleBox">
						 	<span class="el-icon-warning" style="color:#1DA3FC;margin-right:10px;"></span><span>{{titleBox}}</span>
						 </div>
						 <div class="CODE_SYSTEM_BACKUP_RESTORE hidden">  
						 	<el-button v-if="switchFlag == 'off'" type="primary" style="height:28px;width:80px;margin-bottom:16px;margin-left:200px" @click="startBackup(2,'on')"><%=rb.getString("KaiShi")%></el-button>
						 	<el-button v-else type="primary" style="height:28px;width:80px;margin-bottom:16px;margin-left:200px" @click="startBackup(2,'off')"><%=rb.getString("TingZhi")%></el-button>
						 </div>
					  </div>
				</el-form>
				<!-- 分割线 -->
				<div style="height:1px;background:#E8E8E8;"></div>
				<!-- form 表单结束 -->
				<!--备份列表-->
				<!-- <el-progress-table ref="generate" :data="tabledata"></el-progress-table> -->
                    <div style="display:flex;flex-direction:column;flex:1 10%;overflow:auto;">
					<div class="group-title not-extend" style="margin-left:40px;height:65px">
						<span class="title-icon"></span>
						<span class="title-text"><%=rb.getString("BeiFenWenJianLieBiao")%></span>
					</div>
					<el-ctable ref="cbackupTable" :id="'fileTableId'" :url="fileTableUrl" time=6 :pagination=true :query-params="taskParams" height="100%" style="margin-left:40px;width:90%;overflow:auto;">
						<template slot="toolbar">
							<div class="searchCon" >
								<el-input placeholder="<%=rb.getString("WenJianMing")%>" v-model="taskParams.selectText" @keyup.enter.native="searchResult">
									<i slot="suffix" style='margin-top:4px' class="el-icon el-icon-common-search" @click="searchResult"></i>
								</el-input>
							</div>
						</template>
						<el-table-column width="80">
							<template slot-scope="scope">
								<div class="tableDiv operation_zip"></div>
							</template>
						</el-table-column>
						<el-table-column prop="" width="500">
							<template slot-scope="scope" >
								<div class="backupInfo">{{scope.row.fileName}}</div>
								<span style="margin-right:5px;"><%=rb.getString("BeiFenNeiRong")%> :</span><span style="color:#999999">{{scope.row.content}}</span><span style="margin:0px 5px 0px 10px;"><%=rb.getString("BeiFenBanBen")%> :</span><span style="color:#999999">{{scope.row.omcVersion}}</span>
							</template>
						</el-table-column>
						<el-table-column prop="">
							<template slot-scope="scope">
								<div class="backupInfo"><span v-if="scope.row.progress != '100'"><%=rb.getString("BeiFenShiJian")%></span><span v-if="scope.row.progress == '100'"><%=rb.getString("BaoCunShiJian")%></span></div>
								<span style="color:#999999">{{scope.row.startTime}}</span>
							</template>
						</el-table-column>
						<el-table-column>
							<template slot-scope="scope">
								<el-progress stroke-width="8" :percentage="scope.row.progress"></el-progress>
							</template>
						</el-table-column>
	
						<el-table-column>
							<template slot-scope="scope">
								<div v-if="scope.row.status == 0" style="color:#f36666;font-size:14px;"><%=rb.getString("ShiBai")%></div>
								<div v-else-if="scope.row.status == 1" style="color:#c2c2c2;font-size:14px;" ><%=rb.getString("DengDai")%></div>
								<div v-else-if="scope.row.status == 2" style="color:#1da3fc;font-size:14px;" ><%=rb.getString("JinXingZhong")%></div>
								<div v-else style="color:#1da3fc;font-size:14px;" ><%=rb.getString("WanCheng")%></div>
							</template>
						</el-table-column>
						<el-table-column>
							<template slot-scope="scope">
								<!-- <div v-if="scope.row.status == 3" class="el-button--primary btnStyle" @click="downloadFile(scope.row.taskId)"><%=rb.getString("XiaZai")%></div>-->
								<div v-if="false" class="el-button--primary btnStyle" @click="downloadFile(scope.row.taskId)"><%=rb.getString("XiaZai")%></div>
								<div v-if="scope.row.status == 2" class="btnStyle el-button--primary CODE_SYSTEM_BACKUP_RESTORE hidden" @click="cancelDownload(scope.row.taskId)"><%=rb.getString("TingZhi")%></div>
								<div v-else class="btnStyle el-button CODE_SYSTEM_BACKUP_RESTORE hidden" @click="deleteFile(scope.row.taskId)"><%=rb.getString("ShanChu")%></div>
							</template>
						</el-table-column>
					</el-ctable>
				</div>
				<form id="downLoadBackupFile" style="display:none" method="post" action=""></form>
			</el-tab-pane>
			<el-tab-pane label="<%=rb.getString("HuiFu")%>" name="recover" id='recoverPane'>
				<el-ctable :url='fileUrl' ref='recoverTable' :query-params="params" height="100%" page-size="20" pagination=true>
					<template slot="toolbar">
						<div class="searchCon" >
							<el-input @keyup.enter.native="query_recover" v-model='params.selectText' class='pairgrid-query' placeholder='<%=rb.getString("WenJianMing")%>'>
								<i style='margin-top:4px' slot="suffix" class="el-icon el-icon-common-search" @click="query_recover"></i>
							</el-input>
						</div>
					</template>
					<el-table-column label='' width="30">
						<template slot-scope="scope">
	            			<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
	          			</template>
					</el-table-column>
					<el-table-column prop='taskId' v-if=false></el-table-column>
					<el-table-column label='<%=rb.getString("WenJianMing")%>' prop='fileName'></el-table-column>
					<el-table-column label='<%=rb.getString("BeiFenBanBen")%>' width="200" prop='omcVersion'></el-table-column>
					<el-table-column label='<%=rb.getString("BeiFenNeiRong")%>' width="200" prop='content'></el-table-column>
					<el-table-column label='<%=rb.getString("BeiFenShiJian")%>' width="150" prop='startTime'></el-table-column>
				</el-ctable>
				<el-cmenu ref="menu" :data="menus" @click="clickMenu"></el-cmenu>
			</el-tab-pane>
		</el-tabs>
		<el-dialog id='infoDialog' title=' ' custom-class="backRecoverDialogCls" :visible.sync="dialogFormVisible" :width="dialogWidth" top='28vh' :close-on-click-modal=false :show-close=false>
			<!-- 如果是admin用户需要填写密码 -->
			<el-card v-if='step_info'>
				<div slot="header">
					<span><%=rb.getString("TiShi")%></span>
					<span class='el-icon-close' @click='cancelInfo'></span>
				</div>
				<div>
					<el-form :model='fileForm' label-position='top' style='width:300px;margin:20px auto'>
						<el-form-item label='<%=rb.getString("YongHuMingCheng")%>' prop="userName">
							<el-input v-model="fileForm.userName" :disabled="true" style='width:300px;'></el-input>
						</el-form-item>
						<el-form-item label='<%=rb.getString("MiMa")%>' prop="password" style='margin-top:28px;'>
							<el-input type="password" v-model="fileForm.password" style='width:300px;'></el-input>
						</el-form-item>
					</el-form>
					<el-button-group size="mini" style='position:absolute;right:83px;bottom:45px;'>
			 			<el-button type="primary" size="mini" @click='startBackUpInfo'><%=rb.getString("QueDing")%></el-button>
			    		<el-button size="mini" @click='cancelInfo'><%=rb.getString("QuXiao")%></el-button>
			    	</el-button-group>
				</div>
			</el-card>
			<!-- 开始进行备份 -->
			<div class='recoverBg1 backup_start' v-if='step_backup'>
				<span><%=rb.getString("BeiFenZhong")%></span>
				<!-- <span>备份当前环境中的程序、数据、配置等</span> -->
				<p @click='cancelBackup'><%=rb.getString("QuXiao")%></p>
				<p v-loading="loading"></p>
				<p>{{backupContent}}</p>
			</div>
			<!-- 备份失败 -->
			<div class='backupFail' v-if='step_backup_fail'>
				<p class='fail_backup'></p>
				<p><%=rb.getString("BeiFenShiBai")%></p>
				<p><%=rb.getString("BeiFenShiBaiCiPanKongJianBuZu")%></p>
				<p @click='cancelBackupFail'><%=rb.getString("QueDing")%></p>
				<i @click='cancelBackupFail' class="el-icon-close" style='position:absolute;right:20px;top:20px;'></i>
			</div>
			<!-- 备份完成，确认是否开始恢复 -->
			<div class='recoverBg2' v-if='step_sure_recover'>
				<p><%=rb.getString("ShiFouJiXuHuiFu")%></p>
				<p @click='sureRecover'><%=rb.getString("QueDing")%></p>
				<p @click='cancelSureRecover'><%=rb.getString("QuXiao")%></p>
			</div>
			<!-- 开始恢复 -->
			<div class='recoverBg1 recover_start' v-if='step_start_recover'>
				<p><%=rb.getString("HuiFuZhong")%></p>
				<p v-loading="loading"></p>
				<p>Restore is running,please wait!</p>
			</div>
			<!-- 恢复失败 -->
			<div class='backupFail' v-if='step_recover_fail'>
				<p class='fail_backup'></p>
				<p><%=rb.getString("HuiFuShiBai")%></p>
				<p><%=rb.getString("ShiFouChongXinZhiXing")%></p>
				<p style='left:171px;' @click='recoverAgain'><%=rb.getString("ChongXinZhiXing")%></p>
				<p @click='cancelRecoverAgain'><%=rb.getString("QuXiao")%></p>
			</div>
			<!-- 恢复完成 -->
			<div class='backupFail' v-if='step_recover_success'>
				<p class='success_backup'></p>
				<p><%=rb.getString("HuiFuWanCheng")%></p>
				<p>{{recoverSuccess}}</p>
				<!--
				<p @click='startNewSys'><%=rb.getString("QueDing")%></p> -->
			</div>
		</el-dialog>
		
	</template>
</div>
<script>
var backupProTimer;
var startRecoverTimer;
var omc_version = "${omc_ver}"
new Vue({
	el:"#backRecoverPage",
	data(){
		var vm = this;
		
		var validateSetTime = (rule,value,callback) => {
				// dateTimeFlag == true 说明已勾选  校验 用户记录天数是否填写 
				//不为空校验
				if(value == "" && vm.form.resource == '2'){
					callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'))
				}else{
					callback();
				}
			},
			validatePeriod = (rule,value,callback) => {
				if(value == "" && vm.form.resource == '2'){
					callback(new Error('<%=rb.getString("QingXuanZeZhouQi")%>'))
				}else{
					callback();
				}
			};
			
		return {
		    periodOptions:[
				{
					value:'month',
					label:'<%=rb.getString("Yue")%>'
				},
				{
					value:'week',
					label:'<%=rb.getString("Zhou")%>'
				},{
					value:'day',
					label:'<%=rb.getString("Tian")%>'
				}
			],

			activeName:'backup',
			form: {
		          name: '/data/omc/backup',
		          type: ['system_config','system_data'],
		          resource: '1',
		          date1: '',
		          days:'',
				  period: ''
		    },
		    rules:{
		    	date1:[
		    		{validator: validateSetTime}
		    	],
		    	period: [
		    		{validator: validatePeriod}
		    	]
		    },
		    switchFlag:'off',
		    titleBox:'<%=rb.getString("DingShiBeiFenRenWuWeiQiDong")%>',
		    fileTableUrl:'${ctx}/sys/backupAndRecovery/getBackupFileList.action',
		    taskParams:{
				timeZone: timeZone,
				selectText:'',
			},
		    
		   recoverFlag:false,
		   addText:false,
		   buttonIcon:'el-icon-plus',
		   fileUrl:'${ctx}/sys/backupAndRecovery/recoverGetBackupFileList.action',
		   selectText:'',
		   menus:[],
		   fileForm:{
			   userName:'admin',
			   password:''
		   },
		   dialogFormVisible:false,
		   step_info:false,
		   step_backup:false,
		   step_backup_fail:false,
		   step_sure_recover:false,
		   step_start_recover:false,
		   step_recover_fail:false,
		   step_recover_success:false,
		   dialogWidth:'',
		   loading:true,
		   params:{
			   selectText:'',
			   timeZone: timeZone
		   },
		   recoverRow:[],
		   backupContent:'',
		   recoverSuccess:'',
		   rebootFlag:false
		}
	},
	computed:{
		// 根据权限控制表单是否能操作
		formShow() {
			return writableMap['CODE_SYSTEM_BACKUP_RESTORE'] == false;
		},
	},
	methods:{
		// 备份文件列表查询
		searchResult(){
			this.$refs.cbackupTable.refresh();
		},
		/**
		* 备份任务
		* @param typeWay{number}   备份类型 1 立即/  2 定时
		* @param startOrTop{string}   任务类型 off停止任务 on开始任务
		*/ 
		startBackup(typeWay,startOrTop){
			var vm = this; 
			//判断是开始任务还是停止任务
			if(typeWay == 2 && startOrTop == 'off' ){
				var params = {};
				params.timmerSwitch = "off"
				vm.titleBox = '<%=rb.getString("DingShiBeiFenRenWuWeiQiDong")%>';
				vm.switchFlag = "off"
				vm.backupFun(params)
				return ;
			} 
			//如果没有勾选 判断是否都没有勾选    vm.form.type 数组长度为0
			if(vm.form.type.length == 0){
				vm.$message({
					showClose:true,
					message:'select at least one backup',
					type:'error'
				})
				return false;
			}
			//先判断磁盘空间是否可以允许备份
			vm.$refs.backupform.validate(function(valid){
				if(valid){
					var params = {
							timeZone: timeZone
						};
					//增加版本
					params.omcVersion = omc_version;
				    params.backup_way = vm.form.resource;
				    params.backup_item = vm.form.type;
				    if(vm.form.resource == '1'){
				    	
				    }else{
				    	params.start_time = vm.form.date1;
						params.backup_period = vm.form.period;
				    }

					vm.$confirm('<%=rb.getString("BeiFenHaoShiTiShi")%>', '<%=rb.getString("QueRen")%>', {
						confirmButtonText: '<%=rb.getString("QueDing")%>',
						cancelButtonText: '<%=rb.getString("QuXiao")%>',
						type: 'warning',
						dangerouslyUseHTMLString: true 
					}).then(()=>{

						axios.post('${ctx}/sys/backupAndRecovery/checkDiskUsage.action',stringify(params)).then((res)=>{
							let data = res.data || {};

							if(data.msg_target == '0'){//磁盘空间充足 允许备份
								//判断是什么类型的备份
								if(typeWay == 1){//立即备份
									vm.backupFun(params)
								}else{
									//定时备份 定时备份判断第二个参数  是启动还是停止
									
									if(startOrTop == "on"){ //开启定时器
										params.timmerSwitch = "on"
										vm.titleBox = '<%=rb.getString("DingShiBeiFenRenWuJinXingZhong")%>';
										vm.switchFlag = "on"
										vm.backupFun(params);
									}
								}
							
							}else if(data.msg_target == '1' || data.msg_target == '2') {
								var maxFiles = data.max_backup_file_count;
								var msgTips = [
										'<%=rb.getString("CiPanShengYuKongJian")%>' + data.disk_available,
										'<br><%=rb.getString("SuoXuCiPanKongJian")%>' + data.disk_required,
										'<br>' + data.msg_target == '1'? '<%=rb.getString("BeiFenJianShanWenJianTiShi")%>':'<%=rb.getString("BeiFenWenJianGuoDuoTiShi")%>'.replace('two',maxFiles).replace('2',maxFiles)
									].join(' ');

								vm.$confirm(msgTips, '<%=rb.getString("QueRen")%>', {
									confirmButtonText: '<%=rb.getString("QueDing")%>',
									cancelButtonText: '<%=rb.getString("QuXiao")%>',
									type: 'warning',
									dangerouslyUseHTMLString: true 
								}).then(()=>{
									//判断是什么类型的备份
									if(typeWay == 1){//立即备份
										vm.backupFun(params)
									}else{
										//定时备份 定时备份判断第二个参数  是启动还是停止
										
										if(startOrTop == "on"){ //开启定时器
											params.timmerSwitch = "on"
											vm.titleBox = '<%=rb.getString("DingShiBeiFenRenWuJinXingZhong")%>';
											vm.switchFlag = "on"
											vm.backupFun(params);
										}
									}
								}).catch(()=>{});
							}else{
								vm.$message({
									showClose:true,
									message:'<%=rb.getString("CiPanKongJianBuZuQingQingLi")%>',
									type:'error'
								})
							}
						}).catch((error)=>{

						})
					
					}).catch(()=>{});
				}
			})
			
		},
		/**
		* 备份
		* @param params{object}   备份传递的参数
		*/ 
		backupFun(params){
			 axios.post('${ctx}/sys/backupAndRecovery/startBackUp.action',stringify(params)).then((response)=>{
			    	if(true){//备份成功刷新表格
			    		vm.$refs.cbackupTable.refresh();
			    	}else{
			    		//备份失败
			    		
			    	}
			    }).catch((error)=>{
			    	
			    })
		},
		/**
		* 下载备份
		* @param taskId{number}   备份文件id
		*/ 
		downloadFile(taskId){
			var vm = this;
			axios.post('${ctx}/sys/backupAndRecovery/checkFileIsExist.action',stringify(
    				{
    					taskId :taskId
    				}
    			)).then(function(response){
    				let data = response.data;
    				if(data.success){
    					/* $("#downLoadBackupFile").form('submit', {
    						url: "${ctx}/sys/backupAndRecovery/downloadTaskFile.action",
    						onSubmit: function(param) {
    				            param.taskId = taskId;
    							var bool = checkParams(param)
    							if(!bool) return false;
    				        }
    					}); */
    					exportByForm("${ctx}/sys/backupAndRecovery/downloadTaskFile.action",{
    						taskId: taskId
    					});
    				}else
    					vm.$message.error(data["message"]);
    			}).catch(function(error){})
		},
		/**
		* 停止备份
		* @param taskId{number}   备份文件id
		*/ 
		cancelDownload(taskId){
			var vm = this;
			this.$confirm('<%=rb.getString("QueDingTingZhiXiTongBeiFenMa")%>','<%=rb.getString("QueRen")%>',{
				customClass:'warningConfirm',
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(function(){											
				axios.post('${ctx}/sys/backupAndRecovery/stopTask.action',stringify(
    				{
    					taskId :taskId
    				}
    			)).then(function(response){
    				let data = response.data;
    				if(data.success) {
    					vm.$refs.cbackupTable.refresh();
    					vm.$message({
			    			type:'success',
			    			message:'<%=rb.getString("ChengGong")%>'
			    		})
    				}else
    					vm.$message.error(data["message"]);
    			}).catch(function(error){})
			}).catch(function(){})
		},
		/**
		* 删除备份文件
		* @param taskId{number}   备份文件id
		*/ 
		deleteFile(taskId){
			var vm = this;
			this.$confirm('<%=rb.getString("QueRenShanChuWenJian")%>','<%=rb.getString("QueRen")%>',{
				customClass:'warningConfirm',
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(function(){											
				axios.post('${ctx}/sys/backupAndRecovery/deleteTask.action',stringify(
    				{
    					taskId:taskId
    				}
    			)).then(function(response){
    				let data = response.data;
    				if(data.success) {
    					vm.$refs.cbackupTable.refresh();
    					vm.$message({
			    			type:'success',
			    			message:'<%=rb.getString("ChengGong")%>'
			    		})
    				}else
    					vm.$message.error(data["message"]);
    			}).catch(function(error){})
			}).catch(function(){})
		},
		// 初始化配置
		initConfig(){
			//页面初始化时  恢复上次配置
			axios.post('${ctx}/sys/backupAndRecovery/checkAutoTaskStatus.action').then((response)=>{
				var length = Object.keys(response.data).length;//判断返回的是不是为空 如果返回的数据为空 说明上次是立即备份
				if(length == 0){ //立即备份
					 this.form.resource = '1';
				
				}else{ //定时备份
					 this.form.resource = '2';
					 this.switchFlag = response.data.switchFlag;
					 this.form.date1 = response.data.startTime;
					 this.form.type = response.data.content.split(",");
					 this.form.period = response.data.period;
					 if(response.data.switchFlag == "on"){
					 	this.titleBox = '<%=rb.getString("DingShiBeiFenRenWuJinXingZhong")%>';
					 }
				}
			}).catch((error)=>{
				
			})
		},
		//校验数组中是否包含这一项
		testIncludeArray(array,value){
			var testFlag = array.some(function(item,index,array){
				return item === value;
			})
			return testFlag
		},
		// 导航栏切换
		handeClick(tab,event){
			var name = tab.name;
			if(name == 'recover'){
				this.recoverFlag = true;
			}else{
				this.recoverFlag = false;
			}
		},
		// 恢复文件列表查询
	    query_recover(){
	    	this.$refs.recoverTable.refresh();
	    },
		/**
		* 点击更多操作出现菜单
		* @param row{object}   行数据
		* @param ev{object}   event数据
		*/ 
	    optClick(row,event){
	    	var showRecoverFlag = false;
	    	if(is_super_user){
	    		showRecoverFlag = true;
	    	}else{
	    		showRecoverFlag = false;
	    	}
	    	var vm = this;
	   		vm.recoverRow = row;
	    	vm.menus= [
		          {label:'<%=rb.getString("HuiFu")%>',cls:"el-icon el-icon-operation-restore CODE_SYSTEM_BACKUP_RESTORE hidden",show:showRecoverFlag,code:'recover'},
		          {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_SYSTEM_BACKUP_RESTORE hidden",code:'del'},
		    ]
	    	
	    	vm.$nextTick(function(){
	    		document.body.click();
		    	vm.$refs.menu.show(event);
	    	});
	    },
		// 点击页面其他地方菜单收起
	    handerClose(){
	        this.$refs.menu.hide();
	    },
		/**
		* 菜单点击事件
		* @param ev{object}   行数据
		*/ 
	    clickMenu(ev){
	    	var codes = {
	    		recover:this.recoverFile,
	    		del:this.delFile
	    	}
	    	if(codes[ev.code]){
	    		codes[ev.code](this.recoverRow.taskId)
	    	}
	    },
		//点击恢复功能
	    recoverFile(){
	    	var vm = this;
	    	axios.post("${ctx}/sys/backupAndRecovery/checkCanRecovery.action",stringify({
	    		timeZone: timeZone,
	    		taskId:vm.recoverRow.taskId,
	    		omcVersion:omc_version
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			vm.dialogFormVisible = true;
	    	    	vm.dialogWidth = '463px';
	    		    vm.step_info = true;
	    		}else{
	    			vm.$message.error(data.message || '<%=rb.getString("BanBenBuJianRong")%>');
	    		}
	    	})
	    },
		/**
		* 删除恢复文件
		* @param taskId{number}   恢复文件id
		*/ 
	    delFile(taskId){
	    	var vm = this;
	    	vm.$confirm('<%=rb.getString("QueDingShanChuWenJian")%>','<%=rb.getString("QueRen")%>',{
	    		confirmButtonText:'<%=rb.getString("QueRen")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		customClass:'warningConfirm',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post('${ctx}/sys/backupAndRecovery/deleteTask.action',stringify({
	    			taskId:taskId
	    		})).then(function(response){
	    			var data = response.data
	    			if(data["success"]){
	    				vm.$message({
	    					type:'success',
	    					message:'<%=rb.getString("ChengGong")%>'
	    				})
	    				vm.$refs.recoverTable.refresh();
	    			}else{
	    				vm.$message.error(data["message"])
	    			}
	    		})
	    	}).catch()
	    },
		/**
		* 文件导入回调
		* @param response{object}   回调参数
		*/ 
	    importFile(response){
	    	if(response["success"]){
	    		this.$refs.recoverTable.refresh();
	    	}else{
	    		this.$message.error(response["message"]);
	    	}
	    },
		//点击开始备份
	    startBackUpInfo(){
	    	var vm = this;
	    	axios.post("${ctx}/sys/backupAndRecovery/recoverBackupTask.action",stringify({
	    		timeZone: timeZone,
	    		taskId:vm.recoverRow.taskId,
	    		password:vm.fileForm.password
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			vm.getBackupPro();
	    			backupProTimer = setInterval(function(){
	    				vm.getBackupPro()
						var dom = $("#backRecoverPage");
						if(dom.length == 0) clearInterval(backupProTimer);
	    			},5000)
	    		}else{
	    			vm.$message.error(data["message"])
	    		}
	    	})
	    },
		//点击取消输入密码页面
	    cancelInfo(){
	    	var vm = this;
	    	vm.step_info = false;
	    	vm.dialogFormVisible = false;
	    	vm.fileForm.password = '';
	    },
		//获取备份的进度内容
	    getBackupPro(){
	    	var vm = this;
	    	axios.post("${ctx}/sys/backupAndRecovery/getRecoverProgress.action",stringify({
	    		timeZone: timeZone,
				taskId:vm.recoverRow.taskId
			})).then(function(response){
				var data = response.data;
				if(data["success"]){
					if(data.taskProgress == '0'){
						//备份中
						vm.backupContent = data.content;
						vm.step_info = false;
						vm.fileForm.password = '';
						vm.step_backup = true;
					}else if(data.taskProgress == '1'){
						//备份失败
						clearInterval(backupProTimer);
						vm.step_info = false;
						vm.fileForm.password = '';
						vm.step_backup = false;
						vm.step_backup_fail = true;
					}else if(data.taskProgress == '2'){
						//备份完成
						clearInterval(backupProTimer);
						vm.step_info = false;
						vm.fileForm.password = '';
						vm.step_backup = false;
						vm.step_sure_recover = true;
					}
				}
			})
	    },
		//点击关闭备份失败页面
	    cancelBackupFail(){
	    	var vm = this;
	    	vm.step_backup_fail = false;
	    	vm.dialogFormVisible = false;
	    },
		//点击关闭备份中页面
	    cancelBackup(){
	    	var vm = this;
	    	axios.post("${ctx}/sys/backupAndRecovery/confirmRecover.action",stringify({
	    		timeZone: timeZone,
	    		taskId:vm.recoverRow.taskId,
	    		confirmValue:false,
	    		repeatFlag:false
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			clearInterval(backupProTimer);
	    	    	vm.step_backup = false;
	    	    	vm.dialogFormVisible = false;
	    		}else{
	    			vm.$message.error(data["message"]);
	    		}
	    	})
	    	
	    },
		//确定进行恢复
	    sureRecover(){
	    	var vm = this;
	    	axios.post("${ctx}/sys/backupAndRecovery/confirmRecover.action",stringify({
	    		timeZone: timeZone,
	    		taskId:vm.recoverRow.taskId,
	    		confirmValue:true,
	    		repeatFlag:false
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			vm.step_sure_recover = false;
	    			vm.step_start_recover = true;
	    			vm.getRecoverProgress();
	    			startRecoverTimer = setInterval(function(){
	    				vm.getRecoverProgress();
						var dom = $("#backRecoverPage");
						if(dom.length == 0) clearInterval(startRecoverTimer);
	    			},5000)
	    		}
	    	})
	    },
		//点击取消关闭是否恢复页面
	    cancelSureRecover(){
	    	var vm = this;
	    	axios.post("${ctx}/sys/backupAndRecovery/confirmRecover.action",stringify({
	    		taskId:vm.recoverRow.taskId,
	    		confirmValue:false
	    	})).then(function(response){
	    		var data = response.data
	    		if(data["success"]){
	    			vm.step_sure_recover = false;
	    	    	vm.dialogFormVisible = false;
	    		}else{
	    			vm.$message.error(data["message"]);
	    		}
	    	})
	    	
	    },
		//获取恢复进度
	    getRecoverProgress(){
	    	var vm = this;
	    	axios.post("${ctx}/sys/backupAndRecovery/getRecoverProgress.action",stringify({
	    		timeZone: timeZone,
				taskId:vm.recoverRow.taskId
			})).then(function(response){
				var data = response.data;
				if(data["success"]){
					if(data.taskProgress == '3'){
						//恢复中
					}else if(data.taskProgress == '4'){
						//恢复完成
						axios.post("${ctx}/sys/backupAndRecovery/getBackupFileType.action",stringify({
							timeZone: timeZone,
							taskId:vm.recoverRow.taskId
						})).then(function(response){
							var data = response.data;
							if(data["success"]){
								if(data.backupFileType == 'userRecord' || data.backupFileType == 'deviceData'){
									vm.rebootFlag = false;
									vm.recoverSuccess = '<%=rb.getString("HuiFuWanCheng")%>'
								}else{
									vm.rebootFlag = true;
									vm.recoverSuccess = '<%=rb.getString("HuiFuWanChengQingChongQi")%>'
								}
							}
						})
						clearInterval(startRecoverTimer);
						vm.step_start_recover = false;
						vm.step_recover_success = true;
					}else if(data.taskProgress == '5'){
						//恢复失败
						clearInterval(startRecoverTimer);
						vm.step_start_recover = false;
						vm.step_recover_fail = true;
					}
				}
			})
	    },
		//重启系统
	    startNewSys(){
	    	var vm = this;
	    	if(vm.rebootFlag == false){//不需要重启
	    		vm.dialogFormVisible = false;
	    		vm.step_recover_success = false;
	    	}else{//需要重启
	    		axios.post("${ctx}/sys/backupAndRecovery/executeRestart.action");
	    	}
	    },
		//重新执行
	    recoverAgain(){
	    	var vm = this;
	    	axios.post("${ctx}/sys/backupAndRecovery/confirmRecover.action",stringify({
	    		timeZone: timeZone,
	    		taskId:vm.recoverRow.taskId,
	    		confirmValue:true,
	    		repeatFlag:true
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			vm.step_recover_fail = false;
	    			vm.step_start_recover = true;
	    			vm.getRecoverProgress();
	    			startRecoverTimer = setTimeout(function(){
	    				vm.getRecoverProgress();
	    			},5000)
	    		}
	    	})
	    },
		//恢复失败取消接口
	    cancelRecoverAgain(){
	    	var vm = this;
	    	axios.post("${ctx}/sys/backupAndRecovery/rollBackTask.action").then(function(response){
	    	})
	    	vm.step_recover_fail = false;
	    	vm.dialogFormVisible = false;
	    }
	},
	created(){
		this.initConfig();
	}
	
	
})
</script>