<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style type="text/css">
	#cpeRebootTaskContent {
		border: 0;
		background: #F6F7FB;
	}
	#cpeRebootTaskContent .tow-line-split {
		height: 100%;
		display: flex;
		flex-direction: column;
	}
	#cpeRebootTaskContent .line-flex-item {
		flex: auto;
		overflow: auto;
	}
	
	#cpeRebootTaskContent .result-infos {
		display: flex;
		align-items: center;
		height: 16px;
		max-height: 16px;
		margin-left: 10px;
		border-right: 1px solid #D5DCEC;
		font-size: 14px;
	}
	#cpeRebootTaskContent .result-infos:last-child {
		border-right: 0
	}
	#cpeRebootTaskContent .result-infos i {
		margin-right: 5px;
	}
	#cpeRebootTaskContent .result-count {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 100%;
		min-width: 30px;
		text-align: center;
		color: #000;
		background-color: #fff;
	}

	#cpeRebootTaskContent .el-icon.success-color::before,
	#cpeRebootTaskContent .success-color .el-icon-operation-defaultBeta::before,
	#cpeRebootTaskContent .status-info .el-icon-circle-success::before {
		color: #67d972;
	}
	#cpeRebootTaskContent .el-icon.failure-color::before,
	#cpeRebootTaskContent .failure-color .el-icon-circle-close::before,
	#cpeRebootTaskContent .status-info .el-icon-circle-close::before {
		color: #e88282;
	}
	#cpeRebootTaskContent .success-color .el-icon-operation-defaultBeta,
	#cpeRebootTaskContent .failure-color .el-icon-circle-close {
		font-size: 14px;
	}
	#cpeRebootTaskContent .failure-color .el-icon-circle-close:before{
		color:#E88282;
		content:'\e6fb';
	}
	
	#cpeRebootTaskContent .list_item .el-ctable-toolbar,
	#cpeRebootTaskContent .cpeDeviceTable .el-ctable-toolbar {
		padding: 0 !important;
	}
</style>
<div class="overflow-cls">
<!-- cPE重启任务 -->
<div class="panelDefault" id='cpeRebootTaskContent' style="min-width: 1200px; position: relative;">
	<div class="tow-line-split">
		<div class="line-flex-item commonWarp">
			<el-ctable ref="device" class='cpeDeviceTable'
				:url="cpeDeviceURL"
				:query-params="queryParams"
				@selection-change="selectChange">
				<template slot="toolbar">
					<div class='toolbarHeadBtnBoxCls commonQuery' style='border-top: 0; margin-bottom: 0;height:45px;'>
						<p class='commonSize14' style="padding: 0 0px 0 10px; font-weight: 800;"><%=rb.getString("SuoPinCPELieBiao")%></p>
						<div v-if="hasRebootRole" class="newIconBoxCls-bt" style="right:10px;top:10px;" @click="toAddTask" tip="<%=rb.getString("XinZeng")%>">
							<span class='el-icon el-icon-plus'></span>
						</div>
						<el-query type="normal" @query="queryDevice" placeholder="<%=rb.getString("CPEXuLieHao")%> / <%=rb.getString("CPEName")%> / PCI"></el-query>
					</div>
				</template>
				<el-table-column v-if="hasRebootRole" type="selection" :resever-selection="true"></el-table-column>
				<el-table-column prop="connection_status" width="50">
					<template slot-scope="scope">
						<div :class="{
							'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
							'':scope.row.have_connected==2,
							'conn_exc':scope.row.connection_status=='Exception',
							'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("CPEXuLieHao")%>" prop="serial_number"></el-table-column>
				<el-table-column label="<%=rb.getString("CPEName")%>" prop="host_name"></el-table-column>
				<el-table-column label="IMSI" prop="imsi"></el-table-column>
				<el-table-column label="<%=rb.getString("SheBeiXingHaoMing")%>" prop="model_name"></el-table-column>
				<el-table-column label="<%=rb.getString("MACDiZhi")%>" prop="mac_address"></el-table-column>
				<el-table-column label="<%=rb.getString("IPDiZhi")%>" prop="ipaddress"></el-table-column>
				<el-table-column label="PCI" prop="pci"></el-table-column>
				<el-table-column label="<%=rb.getString("SheBeiZu")%>" prop="group_name"></el-table-column>
			</el-ctable>
		</div>
		<div style="padding: 5px;"></div>
		<div class="line-flex-item commonWarp" style="height: 48%;">
			<el-tabs style="height: 100%;" class=" newTabs">
				<el-tab-pane label="<%=rb.getString("RenWuLieBiao")%>">
					<el-ctable ref="taskList" id="cpe_reboot_task_list"
						:time="6"
						:url="taskURL"
						:query-params="taskParams"
						@load-success="taskSuccess">
						<template slot="toolbar">
							<div class='commonFlex commonQuery'>
								<el-query type="normal" @query="queryTask" placeholder="<%=rb.getString("RenWuMingCheng")%>"></el-query>
								<el-date-picker style='margin-left: 20px;' 
									v-model="time"
									type="datetimerange"
									value-format="yyyy-MM-dd HH:mm:ss"
									range-separator="——"  
									@change="dateChange"
									start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
									end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
								</el-date-picker>
							</div>

							<div style="display: flex;position: absolute;right: 0px;top: 15px;">
								<div class="result-infos">
									<i class="el-icon el-icon-status-waiting1"></i> <%=rb.getString("DengDai")%>
									<span class="result-count">{{taskStatistic.WaitingNum}}</span>
								</div>
								<div class="result-infos">
									<i class="el-icon el-icon-status-inProgress"></i> <%=rb.getString("JinXingZhong")%>
									<span class="result-count">{{taskStatistic.ProcessingNum}}</span>
								</div>
								<div class="result-infos">
									<i class="el-icon el-icon-status-terminate"></i> <%=rb.getString("ZhongZhi")%>
									<span class="result-count">{{taskStatistic.SuspendedNum}}</span>
								</div>
								<div class="result-infos">
									<i class="el-icon el-icon-status-success"></i> <%=rb.getString("JieShu")%>
									<span class="result-count">{{taskStatistic.EndNum}}</span>
								</div>
							</div>
						</template>
						<el-table-column width="40">
							<template slot-scope="scope">
								<div class="el-icon el-icon-operation-more" @click="optClick(scope.row, event)"  v-clickoutside="handerClose"></div>
							</template>
						</el-table-column>
						<el-table-column label="<%=rb.getString("RenWuMingCheng")%>" prop="TASK_NAME"></el-table-column>
						<el-table-column label="<%=rb.getString("ChuangJianZhe")%>" prop="CREATE_USER"></el-table-column>
						<el-table-column label="<%=rb.getString("ChuangJianShiJian")%>" prop="CREATE_TIME"></el-table-column>
						<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="TASK_STATUS">
							<template slot-scope="scope">
								<div v-html="taskTableStatus(scope.row.TASK_STATUS)"></div>
							</template>
						</el-table-column>
						<el-table-column label="<%=rb.getString("JinDu")%>" prop="TASK_PROGRESS"></el-table-column>
						<el-table-column label="<%=rb.getString("JieGuo")%>" prop="TASK_RESULT" :formatter="resultFmt"></el-table-column>
						<el-table-column label="<%=rb.getString("KaiShiShiJian")%>" prop="START_TIME"></el-table-column>
						<el-table-column label="<%=rb.getString("JieShuShiJian")%>" prop="END_TIME"></el-table-column>
					</el-ctable>
					
					<el-cmenu ref="menu" :data="menus" @click="clickEvent"></el-cmenu>
				</el-tab-pane>
				<el-tab-pane label="<%=rb.getString("BackupRestoreSheBeiLieBiao")%>">
					<el-ctable ref="deviceList" class='list_item' id="cpe_reboot_device_list"
						:time="6"
						:url="taskDeviceURL"
						:query-params="taskDeviceParams"
						@load-success="taskDeviceSuccess">
						<template slot="toolbar">
							<div class='toolbarHeadBtnBoxCls commonQuery' style='margin-bottom: 0; border-top: 0;height:45px;'>
								<div class="newIconBoxCls-bt" style="right:10px;top:10px;" @click="exportDevice" tip="<%=rb.getString("DaoChu")%>">
									<span class='el-icon el-icon-operation-export'></span>
								</div>
								<el-query @query="queryTaskDevice" placeholder="<%=rb.getString("CPEXuLieHao")%> / <%=rb.getString("CPEName")%>" type="normal"></el-query>
							</div>
							<div style="display: flex;position: absolute;right: 40px;top: 14px;">
								<div style="display: flex;">
									<div class="result-infos success-color">
										<i class="el-icon el-icon-operation-defaultBeta"></i> <%=rb.getString("ChengGong")%>
										<span class="result-count" style="border-color: #67d972;">{{taskDeviceStatistic.SucNum}}</span>
									</div>
									<div class="result-infos failure-color">
										<i class="el-icon el-icon-circle-close"></i> <%=rb.getString("ShiBai")%>
										<span class="result-count" style="border-color: #e88282;">{{taskDeviceStatistic.FailNum}}</span>
									</div>
								</div>
							</div>
						</template>
						<el-table-column label="<%=rb.getString("CPEXuLieHao")%>" prop="SERIAL_NUMBER"></el-table-column>
						<el-table-column label="<%=rb.getString("CPEName")%>" prop="HOST_NAME"></el-table-column>
						<el-table-column label="<%=rb.getString("RenWuMingCheng")%>" prop="TASK_NAME"></el-table-column>
						<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="PROGRESS_STATUS">
							<template slot-scope="scope">
								<div v-html="taskTableStatus(scope.row.PROGRESS_STATUS)"></div>
							</template>
						</el-table-column>
						<el-table-column label="<%=rb.getString("JieGuo")%>" prop="PROGRESS_RESULT" :formatter="resultFmt"></el-table-column>
						<el-table-column label="<%=rb.getString("ShiBaiYuanYin")%>" prop="FAILURE_REASON"></el-table-column>
						<el-table-column label="<%=rb.getString("KaiShiShiJian")%>" prop="START_TIME"></el-table-column>
						<el-table-column label="<%=rb.getString("JieShuShiJian")%>" prop="END_TIME"></el-table-column>
					</el-ctable>
				</el-tab-pane>
			</el-tabs>
		</div>
	</div>

	<!-- 新建、查看、修改任务浮层 -->
	<el-slide ref="slide" class='commonBorderSlide'
		:url="slideUrl" 
		:title="slideTitle" :footer="slideFooter"
        :subloading="slideSubmitLoading"
		@ok='saveRebootTask' @cancel='cancelSlide' 
		ok-text="<%=rb.getString("QueDing")%>" cancel-text="<%=rb.getString("QuXiao")%>">	
	</el-slide>
</div>
</div>
<script type="text/javascript">
	var vmCPEReboot = new Vue({
		el:'#cpeRebootTaskContent',
		data() {

			return {
				menus: [],
				cpeDeviceURL: '${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=3',
				taskURL: '${ctx}/task/reboot/getRebootTaskListForCpe.action',
				taskDeviceURL: '${ctx}/task/reboot/getRebootTaskProgressForCpe.action',

				queryParams: {
					search_text: '',
					timeZone: timeZone,
					isShowSlave: false,
					serial_number: '',
					host_name: '',
					group_id: '',
					productValue: ''
				},
				taskParams: {
					searchText: '',
					timeZone: timeZone,
					startTime: '',
					endTime: '',
					likeFields: 'task_name'
				},
				time: [],
				taskDeviceParams: {
					searchText: '',
					timeZone: timeZone,
					productType: ''
				},

				selectedRows: [],
				taskStatistic: {
					WaitingNum: '0',
					ProcessingNum: '0',
					SuspendedNum: '0',
					EndNum: '0'
				},
				taskDeviceStatistic: {
					SucNum: '0',
					FailNum: '0'
				},

				slideUrl: '',
				slideTitle: '',
				slideFooter: false,
                slideSubmitLoading:'',
			};
		},
		computed: {
			hasRebootRole() {
				return writableMap.CODE_CPE_REBOOT == true;
			}
		},
		methods: {
			optClick(row, evt) {
				var vm = this,
					status = row.TASK_STATUS;

				vm.menus= [
					{label:'<%=rb.getString("XinXi")%>',code:'info',row: row},
					{label:'<%=rb.getString("KaiShi")%>',cls:"CODE_CPE_REBOOT hidden",code:'start',row: row},
					{label:'<%=rb.getString("ZanTing")%>',cls:"CODE_CPE_REBOOT hidden" ,code:'stop',row: row},
					{label:'<%=rb.getString("ZhongZhiRenWu")%>',cls:"CODE_CPE_REBOOT hidden",code:'terminate',row: row},
					{label:'<%=rb.getString("XiuGai")%>',cls:"CODE_CPE_REBOOT hidden",code:'modify',row: row},
					{label:'<%=rb.getString("ShanChu")%>',cls:"CODE_CPE_REBOOT hidden",code:'del',row: row}
				];

				if(!vm.hasRebootRole) {
					vm.menus= [
						{label:'<%=rb.getString("XinXi")%>',code:'info',row: row},
					];
				}

				initTaskStatus(status, vm.menus);
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menu.show(evt);
				});
			},
			handerClose() {
				this.$refs.menu.hide();
			},
			clickEvent(row) {
				var vm = this,
					code = row.code,
					row = row.row,
					codes = {
						info: vm.toViewTask,
						modify: vm.toEditTask,
						start: vm.activeRebootTask,
						stop: vm.suspendRebootTask,
						terminate: vm.terminateRebootTask,
						del: vm.delRebootTask
					};

				if(codes[code]) codes[code](row);
			},
			selectChange(s) {
				this.selectedRows = s;
			},

			resultFmt(row, column, cellValue, index){
				var resultObj = {
						1: '<%=rb.getString("ChengGong")%>',
						2: '<%=rb.getString("BuFenChengGong")%>',
						3: '<%=rb.getString("ShiBai")%>'
					};

				return resultObj[cellValue];
			},
			taskSuccess(data) {
				var vm = this;
				
				if(data && data.properties) {
					Object.assign(vm.taskStatistic, data.properties);
				}
			},
			taskDeviceSuccess(data) {
				var vm = this;
				
				if(data && data.properties) {
					Object.assign(vm.taskDeviceStatistic, data.properties);
				}
			},
			queryDevice(text) {
				var vm = this;

				vm.queryParams.search_text = text;
			},
			queryTask(text) {
				var vm = this;

				vm.taskParams.searchText = text;
			},
			
			queryTaskDevice(text) {
				var vm = this;

				vm.taskDeviceParams.searchText = text;
			},

			toAddTask() {
				var vm = this;

				vm.slideUrl = '${ctx}/task/reboot/goAddTaskForCpe.action';
				vm.slideTitle = '<%=rb.getString("XinZeng")%>';
				vm.slideFooter = true;
                vm.slideSubmitLoading = false;
				vm.$refs.slide.showSlide(function(){
					eventBus.$emit('addRows', vm.selectedRows);
				});
			},
			toEditTask(row) {
				var vm = this;

				vm.slideUrl = '${ctx}/task/reboot/goAddTaskForCpe.action';
				vm.slideTitle = '<%=rb.getString("XiuGai")%>';
				vm.slideFooter = true;
                vm.slideSubmitLoading = false;
				vm.$refs.slide.showSlide(function(){
					eventBus.$emit('modify-task', row, 'edit');
				});
			},
			toViewTask(row) {
				var vm = this;

				vm.slideUrl = '${ctx}/task/reboot/goAddTaskForCpe.action';
				vm.slideTitle = '<%=rb.getString("ChaKan")%>';
				vm.slideFooter = false;
                vm.slideSubmitLoading = false;
				vm.$refs.slide.showSlide(function(){
					eventBus.$emit('modify-task', row, 'view');
				});
			},
			/**
			 * 开始执行任务函数
			 * @param task_id: 当前数据ID
			*/
			activeRebootTask(row){ 
				var vm = this,
					task_id = row.TASK_ID;

				axios.post('${ctx}/task/reboot/activeTaskForCpe.action',stringify({
					taskId : task_id,
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$refs.taskList.refresh();
					}else{
						vm.$message.error(data["message"]);
					}
				})
			},
			/**
			 * 暂停任务函数
			 * @param task_id: 当前数据ID
			*/
			suspendRebootTask(row){
				var vm = this,
					task_id = row.TASK_ID;
				
				axios.post('${ctx}/task/reboot/suspendTaskForCpe.action',stringify({
					taskId : task_id,
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$refs.taskList.refresh();
					}else{
						vm.$message.error(data["message"]);
					}
				})
			},
			/**
			 * 终止任务函数
			 * @param task_id: 当前数据ID
			*/
			terminateRebootTask(row){ 
				var vm = this,
					task_id = row.TASK_ID;

				axios.post('${ctx}/task/reboot/terminateRebootTaskForCpe.action',stringify({
					taskId : task_id,
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$refs.taskList.refresh();
					}else{
						vm.$message.error(data["message"]);
					}
				})
			},
			/**
			 * 软件升级  删除任务
			 * @param task_id: 当前数据ID
			*/
			delRebootTask(row){
				var vm = this,
					task_id = row.TASK_ID;

				this.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>',QueRen,{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post('${ctx}/task/reboot/delRebootTaskForCpe.action',stringify({
						taskId:task_id,
					})).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$refs.taskList.refresh();
							vm.$message({
								type:'success',
								message:'<%=rb.getString("ChengGong")%>'
							})
						}else{
							vm.$message.error(data["message"]);
						}
					}).catch(function(error){
						
					})
				}).catch()
			},
			saveRebootTask() {
				eventBus.$emit('hander-ok');
			},
			cancelSlide() {
				var vm = this;

				vm.$refs.device.clearSelection();
				vm.$refs.taskList.refresh();
				vm.$refs.deviceList.refresh();
				vm.$refs.slide.hide();
			},
			exportDevice() {
				var vm = this,
					url = '${ctx}/task/reboot/exportRebootProgResultForCpe.action',
					params = {
						timeZone: timeZone,
						searchText: vm.taskDeviceParams.searchText
					};

				exportByForm(url, params);
			},
			dateChange(val) {
				var vm = this;
				vm.time = val;
				if(val != null){
					vm.taskParams.startTime = vm.time[0];
					vm.taskParams.endTime = vm.time[1];
				}
			},
		},
		watch:{
			time(newVal){
				var vm = this;
				if(!newVal){
					newVal = [];
					vm.taskParams.startTime = '';
	    			vm.taskParams.endTime = '';
				}
			}
		},
		mounted() {
			eventBus.$on('cancel-reboot',this.cancelSlide);
			eventBus.$on('hide-reboot',this.cancelSlide);
		}
	})
</script>
