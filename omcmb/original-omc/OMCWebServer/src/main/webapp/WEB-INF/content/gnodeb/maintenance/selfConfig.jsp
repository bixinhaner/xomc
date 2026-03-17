<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<!DOCTYPE html>
<html>
<head>
	<style>
		.importPlanClass{
			display:flex;
			flex-direction:column;
			width:580px;
			height:300px;
			background:#fff;
			border:1px solid #DEDFE6;
			position:absolute;
			right:70px;
			top:-100px;
			border-radius:4px;
			box-shadow:0 2px 12px 0 rgba(0,0,0,0.1);
			z-index: 100;
		}
		.importPlanClass .el-input{
			width:400px;
		}
		.color-tip .el-icon-circle-info:before{
			color:#CFCFCF;
		}
		.color-tip .el-icon-common-query-down:before{
			color:#363B4E;
		}
	</style>
</head>
<body>
	<div id="gnb_self_config" class="flex-task-ctn">
		<el-tabs v-model="tabActive" style="height: 100%;" @tab-click="tabClick">
			<el-tab-pane label="Execute Status" name="status">
				<!-- 任务列表 -->
				<el-ctable ref="taskList"
					:url="url"
					:query-params="params_task"
					row-key="task_id">

					<template slot="toolbar">
						<el-query type="normal" placeholder="Serial Number" @query="queryTask"></el-query>
					</template>
					<el-table-column width="40">
						<template slot-scope="scope">
							<div class="el-icon el-icon-operation-more" @click="optClick(scope.row, event)"  v-clickoutside="handerClose"></div>
						</template>
					</el-table-column>
					<el-table-column label="Serial Number" prop="serial_number"></el-table-column>
					<el-table-column label="Progress" prop="execute_procedure"></el-table-column>
					<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="status">
						<template slot-scope="scope">
							<div v-if="scope.row.status == '0'">
								<span class='el-icon el-icon-status-waiting1' style='margin-right:5px;'></span><%=rb.getString("DengDai")%>
							</div>
							<div v-if="scope.row.status == '1'">
								<span class='el-icon el-icon-status-inProgress' style='margin-right:5px;'></span><%=rb.getString("JinXingZhong")%>
							</div>
							<div v-if="scope.row.status == '2'">
								<span class='el-icon el-icon-status-terminate' style='margin-right:5px;'></span><%=rb.getString("YiJieShu")%>
							</div>
						</template>
					</el-table-column>
					<el-table-column label="Results" prop="result">
						<template slot-scope="scope">
							<div v-if="scope.row.result == '0'">
								<span class='el-icon el-icon-status-waiting1' style='margin-right:5px;'></span><%=rb.getString("ChengGong")%>
							</div>
							<div v-if="scope.row.result == '1'">
								<span class='el-icon el-icon-status-inProgress' style='margin-right:5px;'></span><%=rb.getString("ShiBai")%>
							</div>
							<div v-if="scope.row.result == '2'">
								<span class='el-icon el-icon-status-inProgress' style='margin-right:5px;'></span><%=rb.getString("BuFenChengGong")%>
							</div>
						</template>
					</el-table-column>
					<el-table-column label="Start Time" prop="start_time"></el-table-column>
					<el-table-column label="End Time" prop="end_time"></el-table-column>
				</el-ctable>

				<el-cmenu ref="menu" :data="menus" @click="clickHandler"></el-cmenu>
			</el-tab-pane>
			
			<el-tab-pane label="Self-start" name="param" style="background-color: #fff; overflow: auto;">
				<div class="group-title not-extend" style='margin-left:35px;margin-top:40px;'>
					<span class="title-text"></span>
					<p>
						<span><%=rb.getString("ShiFouQiYong") %></span>
						<el-switch style='margin-left:10px;' 
							:disabled="!isWritable"
							v-model="selfForm.selfStartEnable"
							active-value="1"
							inactive-value="0"
							active-color="#4D84FF" 
							inactive-color="#BDC1C6"
							@change="changeSelfConfig"></el-switch>
					</p>
				</div>
				<p style="min-height:1px;background:#E9E9E9;margin-top:40px;"></p>

				<!-- license -->
				<div class="group-title not-extend" style='margin-left:35px;margin-top:40px;'>
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("LicenseXiaFa") %></span>
					<el-switch 
						:disabled="!isWritable"
						v-model="selfForm.licenseEnable"
						active-value="1"
						inactive-value="0"
						active-color="#4D84FF" 
						inactive-color="#BDC1C6"
						@change="changeSelfConfig"></el-switch>
				</div>
				<div style="font-size:12px;margin-left:60px;margin-top:15px;">
					<span class="el-icon el-icon-circle-info" style="font-size:14px"></span>
					<span @click="viewLicense" style="color:#4D84FF;text-decoration:underline;cursor:pointer;"><%=rb.getString("DianJiChaKanLicense") %></span>
				</div>
				<p style="min-height:1px;background:#E9E9E9;margin-top:40px;"></p>
				
				<!-- Parameter Configuration -->
				<div class="group-title not-extend" style='margin-left:35px;margin-top:40px;'>
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("CanShuZiPeiZhi") %></span>
					
					<el-switch 
						:disabled="!isWritable"
						v-model="selfForm.selfConfigEnable"
						active-value="1"
						inactive-value="0"
						active-color="#4D84FF" 
						inactive-color="#BDC1C6"
						@change="changeSelfConfig"></el-switch>
				</div>
				<div style="margin-left:60px;margin-top:15px;width:93%;position:relative">
					<p class="color-tip">
						<span class="el-icon el-icon-circle-info" style="font-size:14px;"></span>
						<span style="color:#BBB"><%=rb.getString("CanShuPeiZhiTiShi") %></span>
					</p>
					<div style="display:flex;margin-top:15px;">
						<label style="flex:1 1 auto;line-height:30px;"><%=rb.getString("WenJianLieBiao")%></label>
						<span style="display:inline-block;margin-bottom:5px;">
							<div class="importPlanClass" v-show="importVisible">
								<div>
									<div class="el-card__header">
										<span ><%=rb.getString("DaoRu")%></span>
										<span class=" el-icon el-icon-close" @click="cancelImport" style="font-size:16px;"></span>
									</div>
									<div class="el-card__body" style="background:#fff;flex:1 1 auto; border: none;">
										<el-form ref="importForm" :rules="rules" :model="importForm" label-position="top" style="margin-left:30px;margin-top:30px;">
											<el-form-item label="<%=rb.getString("WenJian")%>" prop="filePath">
												<el-input v-model="importForm.filePath" :disabled="true" style='width:400px;'>
													<i @click="importFile" slot='suffix' style='display:inline-block;width:28px;height:28px;margin:4px -9px 0 0;' class='el-icon el-icon-operation-import'></i>
												</el-input>
											</el-form-item>

											<p style="color:#BBB">
												<span style='font-size:14px;' class="el-icon el-icon-circle-info"></span>
												<%=rb.getString("ZiQiDongDaoRuWenJianTiShi")%>
											</p>

											<div style="position:relative;margin-top:10px;">
												<p style="cursor:pointer;max-width:160px;" @click="downloadTpl">
													<span class='el-icon el-icon-common-download'></span>
													<span style='text-decoration:underline;cursor:pointe;font-size:14px;'><%=rb.getString("DaoChuMuBan")%></span>
												</p>
											</div>

											<div style='margin-top:45px;'>
												<el-button @click="saveImportFile" type="primary"><%=rb.getString("QueDing")%></el-button>
												<el-button @click="cancelImport"><%=rb.getString("QuXiao")%></el-button>
											</div>
										</el-form>
									</div>
								</div>
							</div>
							
							<span v-show="isWritable" class='el-icon el-icon-circle-import' @click="showImport" style='font-size:24px;margin-right:10px;'></span>

							<span class='el-icon el-icon-circle-export' @click="exportFiles" style='font-size:24px;'></span>
						</span>
					</div>
					<div style="height:300px;border:1px solid #DEDFE6;">
						<el-ctable	ref="ctablePlan" 
							:url="fileURL"
							:query-params="fileParams"
							row-key="id">
							<!-- 模糊查询 -- File List -->
							<template slot="toolbar">
								<el-query type="normal"></el-query>
							</template>
							<el-table-column width="40">
								<template slot-scope="scope">
									<div class="el-icon el-icon-operation-more" @click="optConfigClick(scope.row, event)"  v-clickoutside="closeConfigMenu"></div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serial_number"></el-table-column>
							<el-table-column label="<%=rb.getString("WenJianMing")%>" prop="file_name"></el-table-column>
							<el-table-column label="<%=rb.getString("ShangChuanRen")%>" prop="uploader"></el-table-column>
							<el-table-column label="<%=rb.getString("ShangChuanShiJian")%>" prop="upload_time"></el-table-column>
							<el-table-column label="<%=rb.getString("WenJianDaXiao")%>" prop="file_size"></el-table-column>
						</el-ctable>
						<el-cmenu ref="configMenu" :data="configMenus" @click="fileClickHandler"></el-cmenu>
					</div>
				</div>
			</el-tab-pane>
		</el-tabs>

		<!-- 查看 License -->
		<el-dialog  title='<%=rb.getString("ChaKan")%> License' width='800px' :visible.sync='licenseVisible' :close-on-click-modal="false">
			<div style="height:430px;">
				<el-ctable	ref="ctableLincese" :url="licenseUrl" :query-params="params_license" row-key="id" height="100%">
									
					<!-- 模糊查询 -- 软件升级 -->
					<template slot="toolbar">
						<div class="queryGroup">
							<el-input v-model="params_license_form.search_text" @keyup.enter.native="queryLicense" class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-input>
							<i @click="queryLicense" class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
						</div>
					</template>
					<el-table-column prop="connection_status" width="50">
						<template slot-scope="scope">
							<div :class="{
								'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
								'':scope.row.have_connected==2,
								'conn_exc':scope.row.connection_status=='Exception',
								'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
						</template>
					</el-table-column>
					<el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serial_number"></el-table-column>
					<el-table-column label="<%=rb.getString("LicenseWenJian")%>" prop="file_name"></el-table-column>
					<el-table-column label="<%=rb.getString("ShangChuanShiJian")%>" prop="upload_time"></el-table-column>
					<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="execute_status">
						<template slot-scope="scope">
							<div v-if="scope.row.execute_status == '0'">
								<span class='el-icon el-icon-status-waiting1' style='margin-right:5px;'></span><%=rb.getString("DengDai")%>
							</div>
							<div v-if="scope.row.execute_status == '1'">
								<span class='el-icon el-icon-status-inProgress' style='margin-right:5px;'></span><%=rb.getString("JinXingZhong")%>
							</div>
							<div v-if="scope.row.execute_status == '2'">
								<span class='el-icon el-icon-status-terminate' style='margin-right:5px;'></span><%=rb.getString("YiJieShu")%>
							</div>
						</template>
					</el-table-column>
				</el-ctable>
			</div>
		</el-dialog>

		<el-slide ref="result" position="bottom" 
			height="320px"
			@cancel="closeSlide"
			:footer="false">
			<el-ctable id="exeProgressTable" ref="ctableProgress" 
				:url="taskUrl" 
				:query-params="params_progress" 
				time="6"
				:row-key="'id'" 
				height="100%">
					
				<el-table-column label="<%=rb.getString("JinDu")%>" prop="progress" :formatter="progressFmt"></el-table-column>
				<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="status">
					<template slot-scope="scope">
						<div v-if="scope.row.status == '0'">
							<span class='el-icon el-icon-status-waiting1' style='margin-right:5px;'></span><%=rb.getString("DengDai")%>
						</div>
						<div v-if="scope.row.status == '1'">
							<span class='el-icon el-icon-status-inProgress' style='margin-right:5px;'></span><%=rb.getString("JinXingZhong")%>
						</div>
						<div v-if="scope.row.status == '2'">
							<span class='el-icon el-icon-status-terminate' style='margin-right:5px;'></span><%=rb.getString("YiJieShu")%>
						</div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("JieGuo")%>" prop="result">
					<template slot-scope="scope">
						<div v-if="scope.row.result == '0'">
							<span class='el-icon el-icon-status-waiting1' style='margin-right:5px;'></span><%=rb.getString("ChengGong")%>
						</div>
						<div v-if="scope.row.result == '1'">
							<span class='el-icon el-icon-status-inProgress' style='margin-right:5px;'></span><%=rb.getString("ShiBai")%>
						</div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("KaiShiShiJian")%>" prop="start_time"></el-table-column>
				<el-table-column label="<%=rb.getString("JieShuShiJian")%>" prop="end_time"></el-table-column>
				<el-table-column label="<%=rb.getString("ShiBaiYuanYin")%>" prop="failure_reason"></el-table-column>
			</el-ctable>
		</el-slide>
	</div>

	<!-- 表单-上传基站列表文件 -->
	<form enctype="multipart/form-data" method="post" id="uploadForm_config" style="display: none;">
		<input name="fileSize"  value="" hidden="true">
		<input name="uploadFile" type="file" id="uploadFileConfigPlan">
		<input name="operType" value="">
	</form>
</body>
<script>
	new Vue({
		el: '#gnb_self_config',
		data() {
			var vm = this,
				fileValidator = function(rule, value, cb) {
					if(value) {
						var index = value.lastIndexOf('.'),
							suf = value.substr(index);
						
						if(['.xml','.XML'].includes(suf)) {
							cb();
						}else {
							cb('<%=rb.getString("DangQianZhiChixmlGeShiAll")%>');
						}
					}else {
						cb('<%=rb.getString("DangQianZhiChixmlGeShiAll")%>');
					}
				};
			
			return {
				tabActive: 'status',
				url: '${ctx}/gnb/selfStart/getSelfConfigTaskPageList.action',
				params_task: {
					searchText: '',
					timeZone: timeZone
				},

				selfForm: {
					selfStartEnable: '',
					licenseEnable: '',
					selfConfigEnable: ''
				},
				queryParams: {
					search_text: '',
					monitor: 1,
					TimeZone: timeZone,
					like_fields: 'serial_number,host_name,cell_ip',
					isDual: false,
					isMonitor: true,
					isGnb: 1,
					serial_number: '',
					HOST_NAME: '',
					cell_ip: '',
					op_state: '',
					connection_status: '',
					software_version: '',
					group_id: ''
				},
				menus: [],
				listType: 'task',
				importForm: {
					filePath: ''
				},
				rules: {
					filePath: [{validator: fileValidator}]
				},

				configMenus: [],
				importVisible: false,
				fileURL: '${ctx}/gnb/selfStart/getSelfStartFileInfoData.action',
				fileParams: {
					timeZone: timeZone,
					search_text: ''
				},
				
				licenseVisible: false,
				licenseUrl:"${ctx}/cell/license/getAllLicenseInfoData.action",
				params_license:{
					timeZone: timeZone,
					search_text: '',
					isGnb: '1'
				},
				params_license_form: {search_text: ''},

				taskUrl:"${ctx}/gnb/selfStart/getSelfConfigTaskRecordPageList.action",
				params_progress:{
					timeZone: timeZone,
					task_id: ''
				},
			};
		},
		computed: {
			isWritable() {
				return writableMap['CODE_GNB'] == true;
			}
		},
		methods: {
			tabClick(item) {
				var code = item.name;

				if(code == 'param') {
					this.closeSlide();
				}
			},
			getConfigInfo() {
				var vm = this;

				axios.post("${ctx}/gnb/selfStart/getSelfConfigurationInfo.action").then(function(res){
					var data = res.data;
					if(data){
						Object.assign(vm.selfForm, data);
					}
				}).catch(function(){});
			},
			handerClose() {
				this.hideMenus();
			},
			queryTask(text) {
				this.params_task.searchText = text;
			},
			viewLicense(){
				this.licenseVisible = true;
			},
			queryLicense(){
				Object.assign(this.params_license, this.params_license_form);
			},
			executeStatus(row,column,value,rowIndex){
				if ('0' == value) {
					return "<%=rb.getString("WeiZhiXing")%>";
				} else if ('1' == value){
					return "<%=rb.getString("ZhengZaiZhiXing")%>";
				} else if ('2' == value){
					return "<%=rb.getString("ZhiXingChengGong")%>";
				} else if ('3' == value){
					return "<%=rb.getString("ZhiXingShiBai")%>";
				}
			},
			// 菜单方法 ------>
			optClick(row, evt) {
				var vm = this,
					disabled = row.status == '1';

				vm.menus = [
					{label:'<%=rb.getString("JieGuo")%>',cls:"el-icon el-icon-operation-result",code:'result', row: row},
					{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'delete', row: row, disable: disabled}
				];

		    	vm.$nextTick(function() {
		    		document.body.click();
					vm.showMenus(evt);
		    	});
			},
			clickHandler(item) {
				var vm = this,
					code = item.code,
					row = item.row,
					codes = {
						result: vm.viewResult,
						delete: vm.deleteTask
					};

				if(codes[code]) {
					codes[code](row);
				}
			},
			viewResult(row) {
				var vm = this;

				vm.params_progress.task_id = row.task_id;
				
				vm.$refs.result.showSlide();
			},
			closeSlide() {
				this.$refs.result.hide();
			},
			deleteTask(row) {
				var vm = this;

				vm.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>','<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal: false
				}).then(() => {
					axios.post('${ctx}/gnb/selfStart/delSelfConfigTask.action',stringify({
						taskId: row.task_id,
						timeZone: timeZone
					})).then(function(res){
						var data = res.data;

						if(data.success) {
							vm.$message({
								type: 'success',
								message: '<%=rb.getString("ChengGong")%>'
							});
							vm.$refs.taskList.refresh();
						}else {
							vm.$message({
								type: 'error',
								message: data.msg
							});
						}
					});
				}).catch(function(){});
			},

			optConfigClick(row, evt) {
				var vm = this;

				vm.configMenus = [
					{label:'<%=rb.getString("XiaZai")%>',cls:"el-icon el-icon-operation-download",code:'download', row: row},
					{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_GNB hidden",code:'delete', row: row}
				];

		    	vm.$nextTick(function() {
		    		document.body.click();
					vm.showConfigMenus(evt);
		    	});
			},
			fileClickHandler(item) {
				var vm = this,
					code = item.code,
					row = item.row,
					codes = {
						download: vm.downloadFile,
						delete: vm.deleteFile
					};

				if(codes[code]) {
					codes[code](row);
				}
			},
			downloadTpl() {
				exportByForm('${ctx}/gnb/selfStart/downloadSelfStartTempFile.action',{timeZone: timeZone});
			},
			exportFiles() {
				var vm = this,
					url = '${ctx}/gnb/selfStart/exportSelfStartFileToCSV.action',
					params = {
						search_text: vm.fileParams.search_text,
						timeZone: timeZone
					};

				exportByForm(url,params);
			},
			downloadFile(row) {
				var vm = this,
					url = '${ctx}/gnb/selfStart/downloadSelfStartFile.action',
					params = {
						fileNames: row.file_name
					};

				exportByForm(url,params);
			},
			deleteFile(row) {
				var vm = this;

				vm.$confirm('<%=rb.getString("QueDingShanChuWenJian")%>','<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal: false
				}).then(() => {
					axios.post('${ctx}/gnb/selfStart/clearSelfStartFile.action',stringify({
						fileNames: row.file_name,
						timeZone: timeZone
					})).then(function(res){
						var data = res.data;

						if(data.success) {
							vm.$message({
								type: 'success',
								message: '<%=rb.getString("ChengGong")%>'
							});
							vm.$refs.ctablePlan.refresh();
						}else {
							vm.$message({
								type: 'error',
								message: data.msg
							});
						}
					});
				}).catch(function(){});
			},
			// <------ 菜单方法
			hideMenus() {
				this.$refs.menu.hide();
			},
			closeConfigMenu() {
				this.$refs.configMenu.hide();
			},
			showMenus(evt) {
				this.$refs.menu.show(evt);
			},
			showConfigMenus(evt) {
				this.$refs.configMenu.show(evt);
			},
			addTask() {

			},
			showImport() {
				var vm = this;

				vm.importVisible = true;
				vm.$nextTick(function(){
					vm.$refs.importForm.clearValidate();
				})
			},
			importFile() {
				$("#uploadForm_config input[name='uploadFile']").click();
			},
			cancelImport(){
				this.importVisible = false;
				this.importForm.filePath="";

				$("#uploadForm_config input[name='uploadFile']").val("");
				this.$refs.importForm.resetFields();
			},
			saveImportFile() {
				var vm = this;

				vm.$refs.importForm.validate((valid) => {
					if(valid){
						vm.importLoading = true;
						var files = document.querySelector("#uploadFileConfigPlan").files;

						$("#uploadForm_config [name=fileSize]").val(files[0].size);

						vm.checkExistFile(vm.importForm.filePath, function(){
							uploadWithProgress({
								url: "${ctx}/gnb/selfStart/uploadSelfStartFile.action",
								form: document.querySelector("#uploadForm_config"),
								success: function (data) {
									vm.importLoading = false;
									if(data["success"]){
										vm.cancelImport();
										$("#uploadForm_config input[name='uploadFile']").val("");
										vm.$message({
											message: '<%=rb.getString("ChengGong")%>',
											type:'success'
										})
										vm.$refs.ctablePlan.refresh();
									}else{
										if(data["msg"] == "1"){
											vm.$message.error('<%=rb.getString("DaoRuShiBai")%>')
										}else if(data["msg"] == "2"){
											vm.cancelImport();
											$("#uploadForm_config input[name='uploadFile']").val("");
											vm.$refs.ctablePlan.refresh();
											vm.downloadVisible = true;
										}else{
											vm.$message.error('<%=rb.getString("DaoRuShiBai")%>')
										}
									}
								}
							});
						});
					}
				});
			},
			checkExistFile(fileName, cb) {
				var vm = this,
					rows = vm.$refs.ctablePlan.getData();
				
				axios.post('${ctx}/gnb/selfStart/getAllSelfStartFileInfo.action').then(function(res){
					var data = res.data,
						index = fileName.lastIndexOf('\\'),
						nameStr = fileName.substr(index+1);
						names = nameStr.split('_'),
						sn = names[1]||'',
						existed = false;

					if(data) {
						rows.map(function(row) {
							if(data[sn] == row.file_name) existed = true;
						});

						if(existed) {
							vm.$confirm('<%=rb.getString("ShiFouFuGaiWenJian")%>','<%=rb.getString("QueRen")%>',{
								confirmButtonText:'<%=rb.getString("QueDing")%>',
								cancelButtonText:'<%=rb.getString("QuXiao")%>',
								type:'warning',
								closeOnClickModal: false
							}).then(function(r){
								if(r) cb();
							}).catch(function(){});
						}else {
							cb();
						}
					}else {
						cb();
					}
				}).catch(function(){});
			},
			changeSelfConfig(val){
				var vm = this,
					params = vm.selfForm;

				axios.post("${ctx}/gnb/selfStart/updateSelfConfigurationInfo.action",stringify(params)).then(function(res){
					var data = res.data;
					if(data["success"]){
						
					}else{
						vm.$message.error(data["message"])
					}
				})
			},

			progressFmt(row,column,value,index){
				if(value == 1){
					return "<%=rb.getString("LicenseXiaFa")%>"
				}else if(value == 2){
					return "<%=rb.getString("CanShuZiPeiZhi")%>"
				}else if(value == 3){
					return "Cell active"
				}else {
					return "";
				}
			},
			resultViewFmt(row,column,value,index){
				if(value == 0){
					return "<%=rb.getString("ChengGong")%>"
				}else if(value == 1){
					return "<%=rb.getString("ShiBai")%>"
				}
			}
		},
		mounted() {
			var vm = this;

			vm.getConfigInfo();

			$("#uploadForm_config input[name='uploadFile']").bind("change", function() {
				vm.importForm.filePath = this.value;
			});
		}
	});
</script>
</html>