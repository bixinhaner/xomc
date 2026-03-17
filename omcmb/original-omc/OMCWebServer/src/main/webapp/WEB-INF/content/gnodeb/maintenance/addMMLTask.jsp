<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#addMMLScriptTask_gnb {
	display: flex;
	flex-direction: column;
	height: 100%;
}
#addMMLScriptTask_gnb .ml45{
	margin-left: 45px
}
#addMMLScriptTask_gnb .mt20{
	margin-top: 20px
}
#addMMLScriptTask_gnb .taskInput {
	width: 300px;
	height: 28px;
	line-height: 28px;
}
#addMMLScriptTask_gnb .el-form-item__label {
	line-height:26px;
	width:140px;
	text-align:left;
}
#addMMLScriptTask_gnb .titleStyML {
	margin-left: 20px;
	margin-top: 15px;
}
.gnbMmlDialogStyle .el-icon-circle-info:before{
	color:#CFCFCF;
}
.mml-script-footer-gnb {
	display: flex;
	align-items: center;
	padding: 15px 20px;
	border-top: 1px solid #D5DCEC;
}
.gnb-inline-form-item {
	display: inline-block;
}
.gnb-table-width-border {
	border: 1px solid #E9E9E9;
}
.gnb-normal-padding .el-dialog__body {
	padding: 10px 20px !important;
}
</style>

<!-- 新建/查看 gNodeB MML Script任务 -->
<div id="addMMLScriptTask_gnb">
    <div class="slidebarTitleDiv">
        <div style="display: inline-block;"><%=rb.getString("XinJianRenWu")%></div>
    </div>
    
	<div class="circleIcon placeholder-bt" placeholder="<%=rb.getString("GuanBi")%>" @click="closePage">
		<span class="el-icon el-icon-circle-close"></span>
	</div>

	<el-form style="padding: 20px 30px;flex: auto;overflow: auto;"
		ref="form"
		:model='form'
		:rules="rules"
		size="mini"
	>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item label='<%=rb.getString("RenWuMingCheng")%>' prop='taskName' class="ml45 mt20" label-width="150">
			<el-input maxlength="50" v-model="form.taskName" :disabled="isView" size="mini" class="taskInput" placeholder='<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'></el-input>
		</el-form-item>

		<el-form-item v-show="!isView" label='<%=rb.getString("XuanZeJiaoBen")%>' prop="fileName" class="ml45 mt20" label-width="150">
			<el-input type="hidden" v-model="form.fileName"></el-input>
			<div style="display: flex;align-items: center;">
				<el-upload style="width: 300px;"
					ref="file"
					:multiple="true"
					:on-change="fileChange"
					:show-file-list=false
					:action="uploadFileUrl"
					:data="fileData"
					:file-list="form.fileList"
					name="uploadFile"
					accept=".txt"
					:auto-upload="false">
					<el-input :readonly="true" style="width: 100%;" v-model="form.fileName" placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
						<a slot="append" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>
					</el-input>
					<a slot="trigger" ref="file_up"></a>
				</el-upload>
				<div class='commonFlex' style="margin-left: 10px;">
					<span class='commonNotes12'> ( {{formatTips}} )</span>
				</div>
			</div>
			<div>
				<span class="commonNotes12" style="line-height: 22px;">
					<%=rb.getString("ShiYongMuBanDaoRuTiShi")%>
				</span>
				<span style="cursor: pointer;margin-left: 10px;" @click="downloadTpl">
					<i class="el-icon el-icon-common-download"></i>
					<span class="commonNotes12" style="text-decoration: underline;color: #333;"><%=rb.getString("DaoChuMuBan")%>
					</span>
				</span>
			</div>
		</el-form-item>

		<div class="alarmBottomLine"></div>

		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("XuanZeZhiXingFangShi")%></span>
		</div>
		<el-form-item class="ml45 mt20">
			<div>
				<el-radio v-model="form.status" label="active" :disabled="isView"><%=rb.getString("LiJiZhiXing")%></el-radio>
				<el-radio v-model="form.status" label="suspend" :disabled="isView"><%=rb.getString("GuaQi")%></el-radio>
				<el-radio v-model="form.status" label="timing" :disabled="isView"><%=rb.getString("DingShiZhiXing")%></el-radio>

				<el-form-item class="gnb-inline-form-item" prop="time" style="width: 400px;margin-left: 10px;">
					<el-date-picker style="width: 185px;"
						:disabled="form.status != 'timing' || isView"
						v-model="form.time"
						type="datetime"
						placeholder="Please select time"
						:picker-options="pickerOpts"
						value-format="yyyy-MM-dd HH:mm:ss"
					></el-date-picker>
				</el-form-item>
			</div>
			<div style="margin-top: 15px;">
				<el-radio v-model="form.status" :disabled="isView" label="period"><%=rb.getString("ZhouQiRenWu")%></el-radio>

				<el-form-item class="gnb-inline-form-item" prop="periodStartTime" style="margin-left: 10px;">
					<el-date-picker style="width: 240px;"
						:disabled="form.status != 'period' || isView"
						v-model="dateRange"
						type="daterange"
						placeholder="Please select Date"
						value-format="yyyy-MM-dd"
						:picker-options="pickerOpts"
						@change="dateChange"
					></el-date-picker>
				</el-form-item>
				:
				<el-form-item class="gnb-inline-form-item" prop="periodTime">
					<el-time-picker style="width: 110px;"
						:disabled="form.status != 'period' || isView"
						value-format="HH:mm:ss"
						v-model="form.periodTime"
					></el-time-picker>
				</el-form-item>
			</div>
		</el-form-item>

		<div class="alarmBottomLine"></div>

		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingCeLue")%></span>
		</div>
		<el-form-item class="ml45 mt20">
			<div style='margin: 0;'>
				<div>
					<%=rb.getString("LiXianSheBei")%>
					<el-checkbox v-model="form.offlineRetryEnable" true-label="on" false-label="off" :disabled="isView"></el-checkbox>
					<%=rb.getString("DengDaiSheBeiShangXianChongShi")%>
					<el-form-item class="gnb-inline-form-item" prop="offlineRetryWaitTime">
						<el-input v-model="form.offlineRetryWaitTime" :disabled="isView" type="number" style="width: 80px;"></el-input>
					</el-form-item>
					<%=rb.getString("FenZhongS")%>
				</div>
				<div style="margin:20px 0;">
					<%=rb.getString("ZaiXianSheBei")%>
					<el-checkbox v-model="form.failedRetryEnable" true-label="on" false-label="off" :disabled="isView"></el-checkbox>
					<%=rb.getString("PeiZhiShiBaiChongShi")%>
					<el-form-item class="gnb-inline-form-item" prop="failedRetryCount">
						<el-input v-model="form.failedRetryCount" :disabled="isView" type="number" style="width: 70px;"></el-input>
					</el-form-item>
					<%=rb.getString("JianGeCiShuChongShi")%>
					<el-form-item class="gnb-inline-form-item" prop="failedRetryWaitTime">
						<el-input v-model="form.failedRetryWaitTime" :disabled="isView" type="number" style="width: 70px;"></el-input>
					</el-form-item>
					<%=rb.getString("FenZhongS")%>
				</div>
			</div>
		</el-form-item>
	</el-form>

	<div class="mml-script-footer-gnb" v-if="!isView">
		<div>
			<el-button type="primary" @click="saveTask"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="closePage"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</div>

	<el-dialog top="25vh" width="550" custom-class="gnb-normal-padding"
		:visible.sync="dlShow"
		:append-to-body="true"
	>
		<div slot="title">
			<span style="display: inline-block;padding-top: 10px;font-size: 14px;font-weight: bold;"><%=rb.getString("QueRen")%></span>
			<div @click="exportResult" class="newIconBoxCls-bt" style="right:50px;top:5px;" tip="<%=rb.getString("DaoChu")%>">
				<span class='el-icon el-icon-circle-export'></span>
			</div>
		</div>
		<div style="padding: 5px 0;">
			MML Script contains {{resultList.length}} errors. Are you sure you want to continue?
		</div>
		<el-ctable ref="resultList" class="gnb-table-width-border"
			height="300"
			:data="resultList"
			:rownumber="true"
			:front-pagination="true"
			:pagination="true"
		>
			<el-table-column label='<%=rb.getString("MMLJieGuoHangBiaoTi")%>' prop="line" width="80"></el-table-column>
			<el-table-column label='<%=rb.getString("MMLJieGuoBiaoTi")%>' prop="msg" show-overflow-tooltip></el-table-column>
		</el-ctable>
		<div style="padding: 15px 0 5px;text-align: right;">
			<el-button type="primary" @click="continueSave">Continue</el-button>
			<el-button @click="dlShow = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
</div>

<script type="text/javascript">
new Vue({
	el:'#addMMLScriptTask_gnb',
	data(){
		var vm = this,
			validTaskName = (rule, val, cb) => {
				if(val.trim() || vm.isView) {
					cb()
				}else {
					cb('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>')
				}
			},
			validFileName = (rule, val, cb) => {
				if(val.trim() || vm.isView) {
					cb()
				}else {
					cb('<%=rb.getString("QingXianXuanZeWenJian")%>')
				}
			},
			validTime = (rule, val, cb) => {
				if(val.trim() || vm.form.status != 'timing' || vm.isView) {
					cb()
				}else {
					cb('<%=rb.getString("ZhouQiShiJianTiShi")%>')
				}
			},
			validPeriodTime = (rule, val, cb) => {
				if(val.trim() || vm.form.status != 'period' || vm.isView) {
					cb()
				}else {
					cb('<%=rb.getString("BiTian")%>')
				}
			},
			validPeriodDate = (rule, val, cb) => {
				if(val.trim() || vm.form.status != 'period' || vm.isView) {
					cb()
				}else {
					cb('<%=rb.getString("BiTian")%>')
				}
			},
			validOfflineWaitTime = (rule, val, cb) => {
				if(val || vm.isView) {
					cb()
				}else {
					cb('<%=rb.getString("BiTian")%>')
				}
			},
			validFailCount = (rule, val, cb) => {
				if(val || vm.isView) {
					cb()
				}else {
					cb('<%=rb.getString("BiTian")%>')
				}
			},
			validFailWaitTime = (rule, val, cb) => {
				if(val || vm.isView) {
					cb()
				}else {
					cb('<%=rb.getString("BiTian")%>')
				}
			};

		return {
			isView: false,
			formatTips: '<%=rb.getString("LicenseGeShi")%>'.replace('.lic', '.txt'),

			uploadFileUrl: '',
			fileData: [],
			dateRange: [],
			form: {
				taskName: '${addTaskName}',
				fileName: '',
				uploadFile: '',
				fileList: [],
				status: 'active',
				time: '',
				timeZone: timeZone,
				isGnb: '1',
				periodStartTime: '',
				periodEndTime: '',
				periodTime: '',
				offlineRetryEnable: 'off',
				offlineRetryWaitTime: 60,
				failedRetryEnable: 'off',
				failedRetryCount: 3,
				failedRetryWaitTime: 5
			},
			rules: {
				taskName: [{validator: validTaskName}],
				fileName: [{validator: validFileName}],
				time: [{validator: validTime}],
				periodStartTime: [{validator: validPeriodDate}],
				periodTime: [{validator: validPeriodTime}],
				offlineRetryWaitTime: [{validator: validOfflineWaitTime}],
				failedRetryCount: [{validator: validFailCount}],
				failedRetryWaitTime: [{validator: validFailWaitTime}],
			},
			pickerOpts: {
				disabledDate(time) {
					// 禁止选择过去时间 （可以选今天）
					return time.getTime() < Date.now() - 24*60*60*1000;
				}
			},
			resultList: [],
			dlShow: false
		}
	},
	methods:{
		init(){
			var vm = this;

			var addOrInfoFlag = sessionStorage.getItem('addOrInfoFlag_gnb');
			if(addOrInfoFlag) {
				addOrInfoFlag = addOrInfoFlag.split(',');
			}
			if(addOrInfoFlag && addOrInfoFlag[0] == 'info'){
				vm.isView = true;
				var params = {
						timeZone: timeZone,
						taskId: addOrInfoFlag[1],
						isGnb: 1
					};

				axios.post('${ctx}/task/MMLScript/getMMLScriptDetail.action', stringify(params)).then(function(response){
					var data = response.data;

					if(data){
						var isTiming = data.create_status == "timing",
							isPeriod = data.create_status == "period",
							timeArr = (data.period_end_time || '').split(' ');

						Object.assign(vm.form, {
							taskName: data.task_name,
							fileName: '',
							uploadFile: '',
							fileList: [],
							status: data.create_status,
							time: isTiming? data.start_time:'',
							periodStartTime: isPeriod? data.period_start_time:'',
							periodEndTime: isPeriod? timeArr[0]:'',
							periodTime: isPeriod? timeArr[1]:'',
							offlineRetryEnable: data.offlineRetryEnable,
							offlineRetryWaitTime: data.offlineWaitTime,
							failedRetryEnable: data.failedRetryEnable,
							failedRetryCount: data.failedRetryCount,
							failedRetryWaitTime: data.failedRetryWaitTime
						});
					}
				});
			}else {
				vm.isView = false;
			}
		},
		dateChange(val) {
			var vm = this;

			if(val) {
				vm.form.periodStartTime = val[0];
				vm.form.periodEndTime = val[1];
			}else {
				vm.form.periodStartTime = '';
				vm.form.periodEndTime = '';
			}
		},
		/**
		* 选择文件后，校验格式，并赋值页面显示
		* @param file{object}   文件信息
		* @param fileList{Array}  文件列表
		*/
		fileChange(file, fileList){
			var vm = this,
				fileIndex = file.name.lastIndexOf("."),
				fileType = file.name.substr(fileIndex + 1, file.name.length);
			if(['txt'].indexOf(fileType.toLowerCase()) === -1){
				return false;
			}else{
				let arrList = [];
				let uploadFileList = [];
				if(fileList && fileList.length > 0){
					fileList.map((item)=>{
						arrList.push(item.name);
						uploadFileList.push(item.raw)
					})
				}
				vm.form.fileName = arrList.join(',');
				vm.form.uploadFile = uploadFileList[0];
				vm.form.fileList = uploadFileList;
			}
		},
		// 选择文件
		fileSelect(){
			var vm = this;
			vm.form.fileName = '';
			vm.form.uploadFile = '';
			vm.form.fileList = [];
			vm.$refs.file.clearFiles();
			vm.$refs['file_up'].click();
		},
		closePage() {
			var vm = this;

			vm.dlShow = false;
			// 关闭gnb添加任务面板
			cancelCreateMMLScriptTaskGnb();

			if(sessionStorage.getItem('addOrInfoFlag_gnb')){
				sessionStorage.removeItem('addOrInfoFlag_gnb');
			}
		},
		downloadTpl() {
			var vm = this,
				url = '${ctx}/task/MMLScript/exportMMLTemplete.action?isGnb=1';

			exportByForm(url, {});
		},

		saveTask() {
			var vm = this,
				params = {
					token: omctoken,
					timeZone: timeZone,
					isGnb: '1'
				};

			vm.$refs.form.validate(function(valid) {
				if(valid) {
					Object.assign(params, vm.form);

					vm.uploadFileUrl = '${ctx}/task/MMLScript/addTask.action';
					vm.uploadFiles(vm.uploadFileUrl, params, function(data){
						if (data["success"]) {
							vm.$message({
								message: data["msg"],
								type: 'success',
							});
							vm.closePage();
							$("#MMLScriptTaskList_gnb").datagrid("reload");
						} else {
							if(data['type'] == 'list') {
								// 队列数据展示
								vm.resultList = data['msg'] || [];
								vm.dlShow = true;
							} else {
								vm.$message.error(data["msg"]);
							}
						}
					});
				}
			})
		},
		continueSave() {
			var vm = this,
				params = {
					token: omctoken,
					timeZone: timeZone,
					isGnb: '1',
					doContinue: true
				};

			Object.assign(params, vm.form);

			vm.uploadFileUrl = '${ctx}/task/MMLScript/addTask.action';
			vm.uploadFiles(vm.uploadFileUrl, params, function(data) {
				if (data["success"]) {
					vm.$message({
						message: data["msg"],
						type: 'success',
					});
					vm.closePage();
					$("#MMLScriptTaskList_gnb").datagrid("reload");
				} else {
					vm.$message.error(data["msg"]);
				}
			});
		},
		/*
		* 导入函数
		* url：当前的修改或者添加url
		* params： 所有的from参数
		*/
		uploadFiles(url, params, cb) {
			var vm = this,
				xhr = new XMLHttpRequest(),
				fmd = new FormData();
			if (params) {
				for (var key in params) {
					fmd.append(key, params[key]);
				}
			}
			xhr.onreadystatechange = function () {
				if (this.readyState == 4 && xhr.status == 200) {
					var data = JSON.parse(xhr.responseText);
					if (cb && typeof cb == 'function') cb(data);
				}
			}
			xhr.open('post', url);
			xhr.send(fmd);
		},
		exportResult() {
			var vm = this,
				url = '${ctx}/task/MMLScript/exportMMLErrorMsg',
				params = {
					result: JSON.stringify(vm.resultList)
				};

			exportByForm(url, params);
		}
	},
	mounted(){
		this.init();
	}
})

// 取消新建任务（供外部调用）
function cancelCreateMMLScriptTaskGnb() {
	$("#winAddMMLScriptTask_gnb").slideUp();
}
</script>
