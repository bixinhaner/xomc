<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style type="text/css">
	.tow-line-split {
		height: 100%;
		display: flex;
		flex-direction: column;
	}
	.line-flex-item {
		flex: auto;
		overflow: auto;
		max-height: 50%;
	}	
	.active-line-top .el-tabs__nav {
		margin-left: 10px;
		border: none !important;
	}
	.active-line-top .el-tabs__item {
		height: 28px;
		line-height: 28px;
		color: #666;
		border-top: 2px solid transparent;
	}
	.active-line-top .is-active {
		border-top-color: #4D84FF;
		color: #4D84FF;
		border-left: 1px solid #e9e9e9 !important;
		border-right: 1px solid #e9e9e9;
	}
	.result-info {
		display: flex;
		align-items: center;
		height: 25px;
		max-height: 25px;
		padding-left: 10px;
		margin-right: 15px;
		border: 1px solid #4D84FF;
		border-radius: 5px;
		color: #4D84FF;
		background-color: #f2f6ff;
	}
	.result-info i {
		margin-right: 5px;
	}
	.result-count {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0px 10px;
		margin-left: 10px;
		height: 100%;
		min-width: 30px;
		text-align: center;
		color: #000;
		border-left: 1px solid #4D84FF;
		background-color: #fff;
		border-radius: 0 5px 5px 0;
	}
	.success-color {
		background-color: #eefff3;
	}
	.failure-color {
		background-color: #fef2f2;
	}
	.success-color, 
	.el-icon.success-color::before,
	.success-color .el-icon-operation-defaultBeta::before,
	.status-info .el-icon-circle-success::before {
		color: #67d972;
	}
	.failure-color, 
	.el-icon.failure-color::before,
	.failure-color .el-icon-circle-close::before,
	.status-info .el-icon-circle-close::before {
		color: #e88282;
	}
	.failure-color .el-icon-circle-close {
		font-size: 18px;
	}
	.query-form-cls {
		display: flex;
	}
	.query-form-cls .el-form-item {
		display: inline-block;
		margin-right: 40px;
	}
	.operations{
		position: absolute;
		top: 10px;
		right: 20px;
		z-index: 100;
	}
	.searchWarp{
		display: flex;
		align-items: center;
	}
	.cpeListTitle{
		padding-left: 10px;
	}
	.taskListWarp{
		height: 48%;
	}
	.taskListNumberWarp{
		display: flex;
		position: absolute;
		right: 10px;
		top: 10px;
	}
	.deviceListSearchWarp{
		display: flex;
		justify-content: space-between;
	}
</style>

<!-- cPE恢复出厂 -->
<div class="panelDefault" id='factoryResetTaskContent'>
	<!-- 操作按钮 -->
	<div class="operations">
		<div class="placeholder-bt" placeholder="<%=rb.getString("XinZeng")%>">
			<span class="el-icon el-icon-circle-add" @click="toAddTask"></span>
		</div>
	</div>
	<div class="tow-line-split">
		<div class="line-flex-item">
			<el-ctable ref="cpeList" id="cpeListTable" 
				:url="cpeListUrl"
				:query-params="cpeListParams"
				@selection-change="cpeListSelectChange">
				<template slot="toolbar">
					<div class="searchWarp">
						<h3 class="cpeListTitle"><%=rb.getString("SuoPinCPELieBiao")%></h3>
						<el-query type="normal" @query="cpeListQuery" placeholder="<%=rb.getString("CPEXuLieHao")%> / <%=rb.getString("CPEName")%>"></el-query>
					</div>
				</template>
				<el-table-column type="selection" :resever-selection="true"></el-table-column>
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
				<el-table-column label="<%=rb.getString("ChanPinXingHao")%>" prop="model_name"></el-table-column>
				<el-table-column label="<%=rb.getString("MACDiZhi")%>" prop="mac_address"></el-table-column>
				<el-table-column label="<%=rb.getString("IPDiZhi")%>" prop="ipaddress"></el-table-column>
				<el-table-column label="<%=rb.getString("SheBeiZu")%>" prop="group_name"></el-table-column>
			</el-ctable>
		</div>
		
		<div style="padding: 5px;"></div>
		
		<div class="line-flex-item taskListWarp">
			<el-tabs style="height: 99%;" type="card" class="active-line-top">
				<el-tab-pane label="<%=rb.getString("RenWuLieBiao")%>">
					<el-ctable ref="taskList" :time="6" id="taskListTable"
						:url="taskListUrl"
						:query-params="taskListParams"
						@load-success="taskListLoadSuccess">
						<template slot="toolbar">
								<el-query placeholder="<%=rb.getString("RenWuMingCheng")%>"
									@query="taskListQuery"
									@advance-query="taskListQueryAdvance"
									@reset="taskListReset">
									<template slot="form">
										<el-form ref="taskListForm" label-position="top" class="query-form-cls">
											<el-form-item label="<%=rb.getString("RenWuMingCheng")%>">
												<el-input v-model="taskListQueryForm.searchText"></el-input>
											</el-form-item>
											<el-form-item label="<%=rb.getString("ShiJian")%>">
												<el-date-picker v-model="taskListQueryForm.time" type="datetimerange" value-format="yyyy-MM-dd HH:mm:ss" size="mini"></el-date-picker>
											</el-form-item>
										</el-form>
									</template>
								</el-query>
								<div class="taskListNumberWarp">
									<div class="result-info">
										<i class="el-icon el-icon-status-waiting1"></i> <%=rb.getString("DengDai")%>
										<span class="result-count">{{taskStatistic.WaitingNum}}</span>
									</div>
									<div class="result-info">
										<i class="el-icon el-icon-status-inProgress"></i> <%=rb.getString("JinXingZhong")%>
										<span class="result-count">{{taskStatistic.ProcessingNum}}</span>
									</div>
									<div class="result-info">
										<i class="el-icon el-icon-status-terminate"></i> <%=rb.getString("ZanTing")%>
										<span class="result-count">{{taskStatistic.SuspendedNum}}</span>
									</div>
									<div class="result-info">
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
						<el-table-column label="<%=rb.getString("RenWuMingCheng")%>" prop="TASK_NAME" min-width="260"></el-table-column>
						<el-table-column label="<%=rb.getString("ShiFouBaoLiuPeiZhi")%>" prop="KEEP_CONFIG" :formatter="keepCongigFmt" min-width="190"></el-table-column>
						<el-table-column label="<%=rb.getString("CaoZuoRen")%>" prop="CREATE_USER" min-width="100"></el-table-column>
						<el-table-column label="<%=rb.getString("CaoZuoShiJian")%>" prop="CREATE_TIME" min-width="150"></el-table-column>
						<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="TASK_STATUS" min-width="150">
							<template slot-scope="scope">
								<div v-html="taskTableStatus(scope.row.TASK_STATUS)"></div>
							</template>
						</el-table-column>
						<el-table-column label="<%=rb.getString("JinDu")%>" prop="TASK_PROGRESS" min-width="130"></el-table-column>
						<el-table-column label="<%=rb.getString("JieGuo")%>" prop="TASK_RESULT" :formatter="resultFmt" min-width="150"></el-table-column>
						<el-table-column label="<%=rb.getString("KaiShiShiJian")%>" prop="START_TIME" min-width="150"></el-table-column>
						<el-table-column label="<%=rb.getString("JieShuShiJian")%>" prop="END_TIME" min-width="150"></el-table-column>
					</el-ctable>					
					<el-cmenu ref="menu" :data="menus" @click="clickMenu"></el-cmenu>					
				</el-tab-pane>
				
				<el-tab-pane label="<%=rb.getString("BackupRestoreSheBeiLieBiao")%>">
					<el-ctable ref="deviceList" :time="6"  id="deviceListTable" 
						:url="deviceListUrl"
						:query-params="deviceListParams"
						@load-success="deviceListLoadSuccess">
						<template slot="toolbar">
							<div class="deviceListSearchWarp">
								<el-query @query="deviceListQuery" placeholder="<%=rb.getString("CPEXuLieHao")%> / <%=rb.getString("CPEName")%> / <%=rb.getString("RenWuMingCheng")%>" type="normal"></el-query>
								
								<div style="display: flex; margin-right: 40px;">
									<div class="result-info success-color" style="border-color: #67d972;">
										<i class="el-icon el-icon-operation-defaultBeta"></i> <%=rb.getString("ChengGong")%>
										<span class="result-count" style="border-color: #67d972;">{{taskDeviceStatistic.SucNum}}</span>
									</div>
									<div class="result-info failure-color" style="border-color: #e88282;">
										<i class="el-icon el-icon-circle-close"></i> <%=rb.getString("ShiBai")%>
										<span class="result-count" style="border-color: #e88282;">{{taskDeviceStatistic.FailNum}}</span>
									</div>
									<div class="newIconBoxCls-bt" style="right:15px;top:12px;" @click="exportDevice" tip="<%=rb.getString("DaoChu")%>">
										<span class="el-icon el-icon-circle-export"></span>
									</div>
								</div>
							</div>
						</template>
						<el-table-column label="<%=rb.getString("CPEXuLieHao")%>" prop="SERIAL_NUMBER" min-width="160"></el-table-column>
						<el-table-column label="<%=rb.getString("CPEName")%>" prop="HOST_NAME" min-width="160"></el-table-column>
						<el-table-column label="<%=rb.getString("RenWuMingCheng")%>" prop="TASK_NAME" min-width="220"></el-table-column>
						<el-table-column label="<%=rb.getString("ShiFouBaoLiuPeiZhi")%>" prop="KEEP_CONFIG" :formatter="keepCongigFmt" min-width="190"></el-table-column>
						<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="PROGRESS_STATUS" min-width="120">
							<template slot-scope="scope">
								<div v-html="taskTableStatus(scope.row.PROGRESS_STATUS)"></div>
							</template>
						</el-table-column>
						<el-table-column label="<%=rb.getString("JieGuo")%>" prop="PROGRESS_RESULT" :formatter="resultFmt" min-width="110"></el-table-column>
						<el-table-column label="<%=rb.getString("ShiBaiYuanYin")%>" prop="FAILURE_REASON" min-width="130"></el-table-column>
						<el-table-column label="<%=rb.getString("KaiShiShiJian")%>" prop="START_TIME" min-width="130"></el-table-column>
						<el-table-column label="<%=rb.getString("JieShuShiJian")%>" prop="END_TIME" min-width="130"></el-table-column>
					</el-ctable>
				</el-tab-pane>
			</el-tabs>
		</div>
	</div>

	<!-- 新建、查看、修改任务浮层 -->
	<el-slide class='ignore-border' ref="slide" 
		:url="slideUrl" 
		:modal='slideModal' 
		:title="slideTitle" 
		:footer="slideFooter" 
		:header='slideHeader' 
		:position="slidePosition"
	 	:height="slideHeight" 
	 	:width="slideWidth" >	 	
	 </el-slide>
</div>
<script type="text/javascript">
	var factoryResetTaskVue = new Vue({
		el:'#factoryResetTaskContent',
		data() {
			return {
				menus: [],			
				//cpe list
				cpeListUrl: '${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=3',
				cpeListParams: {
					timeZone: timeZone,
					isShowSlave: false,
					serial_number: '',
					host_name: '',
					search_text: '',
					//以下参数无用
					group_id: '',
					productValue: ''
				},
				selection: [],				
				//task list 
				taskListUrl: '${ctx}/cpe/factoryReset/getTaskListInfo.action',
				taskListQueryForm: {
					searchText: '',
					time: ''
				},
				taskListParams: {
					searchText: '',
					timeZone: timeZone,
					startTime: '',
					endTime: '',
					likeFields: 'task_name'
				},
				taskStatistic: {
					WaitingNum: '',
					ProcessingNum: '',
					SuspendedNum: '',
					EndNum: ''
				},
				//device list
				deviceListUrl: '${ctx}/cpe/factoryReset/getDeviceListInfo.action',
				deviceListParams: {
					searchText: '',
					timeZone: timeZone,
				},
				taskDeviceStatistic: {
					SucNum: '',
					FailNum: ''
				},
				slideUrl:'',
				slideTitle:'',
				slideFooter:'',
				slideHeader:'',
				slidePosition:'',
				slideHeight:'',
				slideWidth:'',
				slideModal:'',
			};
		},
		methods: {
			//cpe list 选择数据
			cpeListSelectChange(selection) {
				this.selection = selection;
			},						
			//task list 表格数据加载成功回调
			taskListLoadSuccess(data) {
				var vm = this;				
				if(data && data.properties) {
					Object.assign(vm.taskStatistic, data.properties);
				}
			},
			//device list 表格数据加载成功回调
			deviceListLoadSuccess(data) {
				var vm = this;				
				if(data && data.properties) {
					Object.assign(vm.taskDeviceStatistic, data.properties);
				}
			},
			//task list 表格操作项
			optClick(row, ev) {
				var vm = this,
					status = row.TASK_STATUS;
					this.$root.rowData = row;
				vm.menus= [					
					{label:'<%=rb.getString("KaiShi")%>',cls:"el-icon el-icon-operation-start",code:'start'},
	                {label:'<%=rb.getString("ZanTing")%>',cls:"el-icon el-icon-operation-awaiting",code:'stop'},
	                {label:'<%=rb.getString("ZhongZhi")%>',cls:"el-icon el-icon-operation-terminate",code:'terminate'},								
	                {label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit",code:'modify'},
					{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
	                {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del'},
				];

				initTaskStatus(status, vm.menus);
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menu.show(ev);
				});
			},
			handerClose() {
				this.$refs.menu.hide();
			},
			clickMenu(ev) {
				var vm = this,
					codes = {	
						start: vm.startTask,
						stop: vm.suspendTask,
						terminate: vm.terminateTask,
						modify: vm.editTask,
						info: vm.viewTask,						
						del: vm.deleteTask
					};

				if(codes[ev.code]){
    	    		codes[ev.code](this.$root.rowData)
    	    	}
			},
			//新建任务
			toAddTask(){
				var vm = this;
				vm.slideHeader = true;
	    	    vm.slideTitle = '<%=rb.getString("XinJianRenWu")%>';
	    	    vm.slideUrl = '${ctx}/cpe/factoryReset/toTaskInfoPage.action';
	    	    vm.slideFooter = false;
	    	    vm.slidePosition = 'top';
	    	    vm.slideHeight = '100%';
	    	    vm.slideWidth = '100%';
	    	    vm.$refs.slide.showSlide(function(){	    	    	
					eventBus.$emit("modify-task", vm.selection, 'add');
	    	    });
			},
			//修改任务
			editTask(row) {
				var vm = this;
				vm.slideHeader = true;
				vm.slideTitle = '<%=rb.getString("XiuGai")%>';
				vm.slideUrl = '${ctx}/cpe/factoryReset/toTaskInfoPage.action?type=modify';				
				vm.slideFooter = false;
				vm.slidePosition = 'top'
	    	    vm.slideHeight = '100%'
	    	    vm.slideWidth = '100%'
				vm.$refs.slide.showSlide(function(){
					eventBus.$emit('modify-task', row, 'modify');
				});
			},
			//查看任务
			viewTask(row) {
				var vm = this;
				vm.slideHeader = true;
				vm.slideTitle = '<%=rb.getString("ChaKan")%>';
				vm.slideUrl = '${ctx}/cpe/factoryReset/toTaskInfoPage.action?type=information';				
				vm.slideFooter = false;
				vm.slidePosition = 'top';
	    	    vm.slideHeight = '100%';
	    	    vm.slideWidth = '100%';
				vm.$refs.slide.showSlide(function(){
					eventBus.$emit('modify-task', row, 'information');
				});
			},
			/**
			 * 开始执行任务函数
			 * @param taskId: 当前数据ID
			*/
			startTask(row){ 
				var vm = this;
				axios.post('${ctx}/cpe/factoryReset/activeTask.action',stringify({
					taskId : row.TASK_ID,
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$refs.taskList.refresh();
						//是否需要提示
						vm.$message({
							type:'success',
							message:'<%=rb.getString("ChengGong")%>'
						})
					}else{
						vm.$message.error(data["message"]);
					}
				})
			},
			/**
			 * 暂停任务函数
			 * @param taskId: 当前数据ID
			*/
			suspendTask(row){
				var vm = this;
				axios.post('${ctx}/cpe/factoryReset/suspendTask.action',stringify({
					taskId : row.TASK_ID,
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$refs.taskList.refresh();
						//是否需要提示
						vm.$message({
							type:'success',
							message:'<%=rb.getString("ChengGong")%>'
						})
					}else{
						vm.$message.error(data["message"]);
					}
				})
			},
			/**
			 * 终止任务函数
			 * @param taskId: 当前数据ID
			*/
			terminateTask(row){ 
				var vm = this;
				axios.post('${ctx}/cpe/factoryReset/terminateTask.action',stringify({
					taskId : row.TASK_ID,
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
				})
			},
			/**
			 *  删除任务
			 * @param taskId: 当前数据ID
			*/
			deleteTask(row){
				var vm = this;
				this.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>',QueRen,{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post('${ctx}/cpe/factoryReset/delTask.action',stringify({
						taskId:row.TASK_ID,
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
		
			//cpe list 搜索
			cpeListQuery(text) {
				var vm = this;
				vm.cpeListParams.search_text = text;
			},
			//task list 搜索
			taskListQuery(text) {
				var vm = this;
				vm.taskListParams.searchText = text;
			},
			//task list 高级搜索
			taskListQueryAdvance() {
				var vm = this,
					times = vm.taskListQueryForm.time;
				vm.taskListParams.searchText = vm.taskListQueryForm.searchText;
				vm.taskListParams.startTime = times[0];
				vm.taskListParams.endTime = times[1];
			},
			//task list 重置
			taskListReset() {
				var vm = this;
				Object.assign(vm.taskListQueryForm, {
					searchText: '',
					time: []
				})
			},
			//device list 搜索
			deviceListQuery(text) {
				var vm = this;
				vm.deviceListParams.searchText = text;
			},
			// device list 导出
			exportDevice() {
				var vm = this,
					url = '${ctx}/cpe/factoryReset/exportDeviceListResult.action',
					params = {
						timeZone: timeZone,
						searchText: vm.deviceListParams.searchText
					};
				exportByForm(url, params);
			},
			//结果字段格式化
			resultFmt(row, column, cellValue, index){
				var resultObj = {
						1: '<%=rb.getString("ChengGong")%>',
						2: '<%=rb.getString("BuFenChengGong")%>',
						3: '<%=rb.getString("ShiBai")%>'
					};
				return resultObj[cellValue];
			},
			//是否保留配置格式化
			keepCongigFmt(row, column, cellValue, index){
				var keepCongigObj = {
						0: '<%=rb.getString("Fou")%>',
						1: '<%=rb.getString("Shi")%>'
					};
				return keepCongigObj[cellValue];
			},
			//关闭slide,更新表格数据
			cancelSlide() {
				var vm = this;
				vm.$refs.cpeList.clearSelection();
				vm.$refs.taskList.refresh();
				vm.$refs.deviceList.refresh();
				vm.$refs.slide.hide();
			},			
		},
		mounted() {
			eventBus.$off('hide-slide').$on('hide-slide',this.cancelSlide);
		}
	})
</script>
