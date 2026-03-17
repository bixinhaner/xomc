<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#cpeChangePassWordPage .changePassWordMainBox{
		height: 100%;
		flex: 1;
		display: flex;
		flex-direction: column;
	}
	#cpeChangePassWordPage .container .cmenu{
		z-index: 361!important;
	}
	#cpeChangePassWordPage .deviceTable{
		height: 50%;
	}
	
	#cpeChangePassWordPage .changePassWordTaskBox{
		box-sizing: border-box;
		flex: 1;
		background-color: #FFFFFF;
		overflow: auto;
		margin-top: 10px;
	}
	#cpeChangePassWordPage .taskHeadBox{
		position: absolute;
		top:15px;
		right: 0px;
		display: flex;
		z-index: 99;
	}
	#cpeChangePassWordPage .taskHeadBox  .statisticsDiv{
		height: 16px;
		line-height: 16px;
		display: flex;
		overflow: hidden;
		font-size: 14px;
	}
	#cpeChangePassWordPage .statisticsDiv > div:first-child{
		padding: 0px 0 0 10px;
		color: #4D84FF;
		background: #FFFFFF;
		height: 16px;
		line-height: 16px;
	}
	#cpeChangePassWordPage .statisticsDiv > div:last-child{
		padding: 0px 10px;
	}
	#cpeChangePassWordPage .statisticsDiv .el-icon, #cpeChangePassWordPage .statisticsSuccessDiv .el-icon, #cpeChangePassWordPage .statisticsFailDiv .el-icon{
		margin-right: 6px;
		font-size: 14px;
	}
	#cpeChangePassWordPage .resultHeadBox{
		position: absolute;
		top:17px;
		right: 60px;
		display: flex;
		z-index: 99;
	}
	#cpeChangePassWordPage .resultHeadBox .statisticsSuccessDiv{
		height: 14px;
		line-height: 14px;
		display: flex;
		font-size: 14px;
		border-right: 1px solid #DFE2EE;
	}
	#cpeChangePassWordPage .statisticsSuccessDiv .el-icon::before{
		color:#67D972;
	}
	#cpeChangePassWordPage .statisticsSuccessDiv > div:first-child{
		padding: 0px 10px;
		color: #67D972;
	}
	#cpeChangePassWordPage .statisticsSuccessDiv > div:last-child{
		padding: 0px 10px 0 0;
	}
	#cpeChangePassWordPage .resultHeadBox .statisticsFailDiv{
		height: 14px;
		line-height: 14px;
		display: flex;
		font-size: 14px;
	}
	#cpeChangePassWordPage .statisticsFailDiv .el-icon::before{
		color: #E88282;
		content:'\e6fb';
	}
	#cpeChangePassWordPage .statisticsFailDiv > div:first-child{
		padding: 0px 10px;
		color: #E88282;
	}
</style>
<!-- cpe修改密码 -->
<div class="overflow-cls">
<div class="pageDefault" id='cpeChangePassWordPage' style="min-width: 1200px;width: 100%; background:unset; border: 0;">
	<div class="container">
		<div class="changePassWordMainBox">
			<!-- 操作按钮 -->
			<div class="circleIcon placeholder-bt CODE_CPE_CHANGE_PASSWORD hidden" placeholder="<%=rb.getString("XinJianRenWu")%>" @click="addTask" style='right: 22px; top: 13px;'>		
				<span class="el-icon el-icon-circle-add"></span>
			</div>
		
			<div class="deviceTable commonWarp" style='height: calc(50% - 5px)'>
				<el-ctable
					:url="deviceTableUrl"
					:query-params="queryDeviceParams" 
					ref="changePasswordDeviceTable" 
					:height="height" 
					@selection-change='deviceSelect'
					:page-size="pageSize" 
					:page-list="pageList" 
					pagination="true">
						<!-- 列表toolbar -->
					<template slot="toolbar">
						<div class='commonQuery commonFlex'>
							<span class="commonText14" style='padding: 4px 0 0 10px'><%=rb.getString("SuoPinCPELieBiao")%></span>
							<el-query type="normal" @query="queryDevice" placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%> / PCI"></el-query>
						</div>
					</template>
						<!-- 列表columns -->
					<el-table-column v-if="hasChangeRole" type="selection" width="45"></el-table-column>
					<el-table-column prop="connection_status" width="50">
						<template slot-scope="scope">
							<div :class="{
								'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
								'':scope.row.have_connected==2,
								'conn_exc':scope.row.connection_status=='Exception',
								'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
						</template>
					</el-table-column>
					<el-table-column prop="small_cell_code" v-if="false"></el-table-column>
					<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" min-width="150"></el-table-column>
					<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>" min-width="250" ></el-table-column>
					<el-table-column prop="mac_address" show-overflow-tooltip label="<%=rb.getString("CPEMacAddress")%>"  min-width="140"></el-table-column>
					<el-table-column prop="imsi" show-overflow-tooltip label="<%=rb.getString("IMSI")%>"  min-width="140"></el-table-column>
					<el-table-column prop="pci" label="PCI"  min-width="140"></el-table-column>
					<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="190" ></el-table-column>
				
				</el-ctable>
			</div>
			<div class="changePassWordTaskBox commonWarp" style='height: calc(50% - 5px)'>
				<el-tabs v-model="activeName" class=" newTabs" style='height: 100%;'>
					<el-tab-pane label="<%=rb.getString("RenWuLieBiao")%>" name="task">
						<div class="taskHeadBox">
							<div class="statisticsDiv">
								<div><span class="el-icon el-icon-status-waiting1"></span><%=rb.getString("DengDai")%></div>
								<div>{{statisticTaskStatus.waiting}}</div>
							</div>
							<div class="statisticsDiv">
								<div><span class="el-icon el-icon-status-inProgress"></span><%=rb.getString("JinXingZhong")%></div>
								<div>{{statisticTaskStatus.inProgress}}</div>
							</div>
							<div class="statisticsDiv">
								<div><span class="el-icon el-icon-status-suspend"></span><%=rb.getString("ZanTing")%></div>
								<div>{{statisticTaskStatus.suspend}}</div>
							</div>
							<div class="statisticsDiv">
								<div><span class="el-icon el-icon-status-terminate"></span><%=rb.getString("JieShu")%></div>
								<div>{{statisticTaskStatus.end}}</div>
							</div>
						</div>
						<el-ctable id="changePasswordTaskTable" time=6  @load-success="taskTableLoadSuccess" ref="changePasswordTaskListTable" :url="taskListTableUrl" :query-params="queryTaskParams" :page-size="20" :pagination=true height="100%" class='commonQueryToolbar'>
							<template slot="toolbar">
								<div class='toolbarHeadBtnBoxCls commonQuery' style=" padding: 0 0 10px 0;">
									<el-query type="normal" @query="queryTask" placeholder="<%=rb.getString("RenWuMingCheng")%>"></el-query>
									<el-date-picker style='margin-left: 10px;' 
										v-model="dateValue"
										type="datetimerange"
										value-format="yyyy-MM-dd HH:mm:ss"
										range-separator="——"  
										@change="dateChange"
										start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
										end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
									</el-date-picker>
								</div>
							</template>							
							<el-table-column label="" width="30" class-name="no-text-tips">
								<template slot-scope="scope"><!-- 将元素或组件表示为作用域插槽          -->
									<div class="el-icon el-icon-operation-more" @click="taskOptClick(scope.row,event)" v-clickoutside="hideMenus"></div>
								</template>
							</el-table-column>
							<el-table-column prop="taskName" show-overflow-tooltip label="<%=rb.getString("RenWuMingCheng")%>"  min-width="200"></el-table-column>
							<el-table-column prop="user" label="<%=rb.getString("CaoZuoRen")%>"  min-width="100" ></el-table-column>
							<el-table-column prop="operationTime" label="<%=rb.getString("CaoZuoShiJian")%>" min-width="180"></el-table-column>
							<el-table-column prop="status" label="<%=rb.getString("ZhuangTai")%>" min-width="120">
								<template slot-scope="scope">
									<div v-html="changePasswordTaskTableStatus(scope.row.status)"></div>
								</template>
							</el-table-column>
							<el-table-column prop="progress" label="<%=rb.getString("JinDu")%>" min-width="100"></el-table-column>
							<el-table-column prop="result" label="<%=rb.getString("JieGuo")%>" min-width="100" :formatter="taskTableResult"></el-table-column>
							<el-table-column prop="startTime" label="<%=rb.getString("KaiShiShiJian")%>" min-width="180"></el-table-column>
							<el-table-column prop="endTime" label="<%=rb.getString("JieShuShiJian")%>" min-width="180"></el-table-column>
						</el-ctable>
						<el-cmenu ref="taskMenu" :data="taskMenus" @click="taskMenuClick"></el-cmenu>
					</el-tab-pane>
					<el-tab-pane label="<%=rb.getString("SheBeiLieBiao")%>" name="device"> 
						<div class="resultHeadBox">
							<div class="statisticsSuccessDiv">
								<div><span class="el-icon el-icon-circle-success"></span><%=rb.getString("ChengGong")%></div>
								<div>{{statisticDeviceResult.success}}</div>
							</div>
							<div class="statisticsFailDiv" >
								<div><span class="el-icon el-icon-circle-close"></span><%=rb.getString("ShiBai")%></div>
								<div>{{statisticDeviceResult.fail}}</div>
							</div>							
						</div>
						<div class="newIconBoxCls-bt" placeholder="<%=rb.getString("DaoChu")%>" style='right: 10px; top: 10px;'>
							<span class="el-icon el-icon-circle-export" @click="exportResultTable"></span>
						</div>
						<!--:url="queryResultUrl"  :data-->
						<el-ctable 
							id="changePasswordResultTable" class='commonQueryToolbar'
							ref="queryResultTable" 
							:url="queryResultUrl" 
							:query-params="queryResultParams"
							@load-success="deviceTableLoadSuccess"
							height="100%"
							time=6
							:page-size="20" 
							:pagination=true>
							<template slot="toolbar">
								<div class='toolbarHeadBtnBoxCls commonQuery' style="padding: 0 0 10px 0;">
									<el-query type="normal" @query="queryCpeUpgradeResult" placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%> / <%=rb.getString("RenWuMingCheng")%>"></el-query>
								</div>
							</template>
							<el-table-column prop="taskId" v-if="false"></el-table-column>
							<el-table-column prop="serialNumber" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" min-width="160"></el-table-column>
							<el-table-column prop="cpeName" show-overflow-tooltip label="<%=rb.getString("CPEName")%>"  min-width="100"></el-table-column>
							<el-table-column prop="taskName" show-overflow-tooltip label="<%=rb.getString("RenWuMingCheng")%>"  min-width="280"></el-table-column>
							<el-table-column prop="status" label="<%=rb.getString("ZhuangTai")%>" min-width="120">
								<template slot-scope="scope">
									<div v-html="resultTableStatus(scope.row.status)"></div>
								</template>
							</el-table-column>
							<el-table-column prop="result" label="<%=rb.getString("JieGuo")%>" min-width="110" :formatter="resultTableResult"></el-table-column>
							<el-table-column prop="failureReason" show-overflow-tooltip label="<%=rb.getString("PCILOCKShiBaiYuanYin")%>" min-width="120"></el-table-column>
							<el-table-column prop="startTime" label='<%=rb.getString("KaiShiShiJian")%>' min-width="160" ></el-table-column>
							<el-table-column prop="endTime" label='<%=rb.getString("JieShuShiJian")%>'  min-width="160"></el-table-column>
						</el-ctable>
					</el-tab-pane>
				</el-tabs>
			</div>
		</div>
		<!-- slide -->
		<el-slide ref="changePasswordSlide" id="changePasswordSlide" :url='slideUrl' :title="slideTitle" :footer="slideFooter" header="true" :position="slidePosition" class='commonBorderSlide'
			:height="slideHeight"  :width='slideWidth' :subloading="slideSubmitLoading" @ok="submitSlide"  @cancel="closeSlide" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>
		
    </div>
</div>
</div>
<script type="text/javascript">
var cpeChangePasswordVue = new Vue({
	el:'#cpeChangePassWordPage',
	data(){
		return {
			deviceTableUrl:"${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=3",
			productValue: "1", //产品类型
			selection:[],
			
			queryDeviceParams:{
				searchText:'',
				timeZone:timeZone,
			},
			taskListTableUrl:'${ctx}/task/cpe/changepwd/getCPEChangePasswordList.action',
			activeName:'task',
			queryTaskParams:{
				searchText:'',
				timeZone:timeZone,
				taskName:'',
				startTime:'',
				endTime:'',
			},
			timeRange:[],
			queryTaskForm:{
				taskName:'',
				startTime:'',
				endTime:'',
			},
			queryResultUrl:'${ctx}/task/cpe/changepwd/getDeviceList.action',
			queryResultParams:{
				timeZone: timeZone,
				searchText:'',
			},
			slideUrl:'',
			slideTitle:'',
			slideHeader:'',
			slideFooter:'',
			slidePosition:'',
			slideHeight:'',
			slideWidth:'',
            slideSubmitLoading:'',

            height:'100%',
            pageSize:100,
			pageList:[50,100,200,500],
			taskMenus:[],
            rowData:'',
			taskRowData:'',
			statisticTaskStatus:{
				waiting: 0,
				inProgress: 0,
				suspend: 0,
				end: 0
			},
			statisticDeviceResult:{
				fail: 0,
				success: 0
			},
			dateValue:[],
		}
	},
    computed:{
		isSuperAdmin() {
			return is_super_user == 'true';
		},
		hasChangeRole() {
			return writableMap.CODE_CPE_CHANGE_PASSWORD == true;
		}
    },
	watch:{
		dateValue(newVal){
			var vm = this;
			if(!newVal){
				newVal = [];
				vm.queryTaskParams.startTime = '';
    			vm.queryTaskParams.endTime = '';
			}
		}
	},
	methods:{
		// 任务表格 加载成功回调
		taskTableLoadSuccess(data){
			var vm = this;

			Object.assign(vm.statisticTaskStatus, data.properties);
			
		},
		// 设备表格 加载成功回调
		deviceTableLoadSuccess(data){
			var vm = this;

			Object.assign(vm.statisticDeviceResult, data.properties);
		},
		// 新建设备升级任务
		addTask(){
			var vm = this;
			vm.slideUrl = '${ctx}/task/cpe/changepwd/goAddTask.action';
			vm.slideFooter = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("XinJianMiMaXiuGaiRenWu")%>';
			
			vm.$refs.changePasswordSlide.showSlide(()=>{
				eventBus.$emit('action-init','','add');
			})
		},
		// 设备表格选择事件
		deviceSelect(selection){
			var vm = this;

			vm.selection = selection
		},
		// 结果 导出
		exportResultTable(){
			var vm = this;

			exportByForm("${ctx}/task/cpe/changepwd/exportDeviceListToCSV.action",{
				timeZone: timeZone,
				searchText: vm.queryResultParams.searchText
			});
		},
		// cpe 升级任务结果 表格模糊查询
		queryCpeUpgradeResult(val){
			var vm = this;
			vm.queryResultParams.searchText= val;
		},
		/**
		 *  任务结果转换
		 * @param cellValue:传入的 结果 数据 进行转换
		*/
		taskTableResult(row,column,cellValue,index){
			var resultObj = {
				"1" : "<%=rb.getString("ChengGong")%>",
				"2" : "<%=rb.getString("BuFenChengGong")%>",
				"3" : "<%=rb.getString("ShiBai")%>",
				"" : "",
			}
			return resultObj[cellValue];
		},
		// 设备结果格式化
		resultTableResult(row,column,cellValue,index){
			var resultObj = {
					"1" : '<%=rb.getString("ChengGong")%>',
					"2" : '<%=rb.getString("ZhongZhi")%>',
					"3" : '<%=rb.getString("ShiBai")%>',
					"" : ''
			}
			return resultObj[cellValue];
		},
		// 设备表格 模糊查询
		queryDevice(val){
			var vm = this;
			vm.queryDeviceParams.searchText= val;
		},
		// 升级任务表格 模糊查询
		queryTask(val){
			var vm = this;

			vm.queryTaskParams.searchText= val;
		},
		
		// 任务表格 打开操作菜单
		taskOptClick(row,evt){
			var vm = this;

			vm.taskRowData = row;
			var status = row.status;
			vm.taskMenus = [
                {label:'<%=rb.getString("KaiShi")%>',cls:"el-icon el-icon-operation-start CODE_CPE_CHANGE_PASSWORD hidden",code:'start'},
                {label:'<%=rb.getString("ZanTing")%>',cls:"el-icon el-icon-operation-awaiting CODE_CPE_CHANGE_PASSWORD hidden",code:'wait'},
                {label:'<%=rb.getString("ZhongZhi")%>',cls:"el-icon el-icon-operation-terminate CODE_CPE_CHANGE_PASSWORD hidden",code:'end'},
				{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
				{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_CPE_CHANGE_PASSWORD hidden",code:'mod'},
                {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_CPE_CHANGE_PASSWORD hidden",code:'del'},
            ];

			if(!vm.hasChangeRole) {
				vm.taskMenus = [
					{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
				];
			}

			initTaskStatus(status,vm.taskMenus);
            vm.$nextTick(function(){
                document.body.click();
                vm.$refs.taskMenu.show(evt)
            })
		},
		/**
		* 菜单点击事件
		* @param ev{object}   行数据
		*/ 
		taskMenuClick(evt){
			var vm = this;
			var codes = {
				start:this.taskStart,	// 开始
				wait:this.taskStop,	// 暂停
				end:this.taskTerminate,	// 终止
	    		info:this.taskInfo,  // 详情
	    		mod:this.taskModify,	// 修改
	    		del:this.taskDel,	// 删除
	    	};
			if(codes[evt.code]){
				codes[evt.code]();
			}
		},
		/**
		 * 激活任务
		 * @param id:数据id
		 * @param fileType: 类型
		*/
		taskStart(id,type){
			var vm = this;
			axios.post('${ctx}/task/cpe/changepwd/activeCPEChangePassword.action',stringify({
	    		taskId: vm.taskRowData.taskId,
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.changePasswordTaskListTable.refresh()
	    		}else{
	    			vm.$message.error(data["message"]) //错误提示信息
	    		}
	    	}) 
		},
		/**
		 * 暂停任务
		 * @param id:数据id
		 * @param fileType: 类型
		*/
		taskStop(){
			var vm = this;
			axios.post('${ctx}/task/cpe/changepwd/suspendCPEChangePassword.action',stringify({
	    		taskId: vm.taskRowData.taskId,
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.changePasswordTaskListTable.refresh()
	    		}else{
	    			vm.$message.error(data["message"]) //错误提示信息
	    		}
	    	}) 
		},
		/**
		 * 终止任务
		 * @param id:数据id
		 * @param fileType: 类型
		*/
		taskTerminate(){
			var vm = this;
			axios.post('${ctx}/task/cpe/changepwd/terminateCPEChangePassword.action',stringify({
	    		taskId: vm.taskRowData.taskId,
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.changePasswordTaskListTable.refresh()
	    		}else{
	    			vm.$message.error(data["message"]) //错误提示信息
	    		}
	    	}) 
		},
		/**
		 * 查看任务
		 * @param id:当前数据id
		*/
		taskInfo(){
			var vm = this;
			vm.slideUrl = '${ctx}/task/cpe/changepwd/goAddTask.action';
			vm.slideFooter = false;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("XinXi")%>';
			vm.$refs.changePasswordSlide.showSlide(()=>{
				eventBus.$emit('action-init',vm.taskRowData.taskId,'view');
			})
		},
		/**
		 * 修改任务
		 * @param id:当前数据id
		*/
		taskModify(id){
			var vm = this;
			vm.slideHeader = true;
			vm.slideUrl = '${ctx}/task/cpe/changepwd/goAddTask.action';
			vm.slideFooter = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("XiuGai")%>';
			vm.$refs.changePasswordSlide.showSlide(()=>{
				eventBus.$emit('action-init',vm.taskRowData.taskId,'edit');
			})
		},
		/**
		 * 删除升级任务
		 * @param id:当前数据id
		*/
		taskDel(){
			var vm = this,
				params = {
					taskId: vm.taskRowData.taskId,
				};
			vm.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>','<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(function(){
				axios.post('${ctx}/task/cpe/changepwd/deleteCPEChangePassword.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data) {
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});
							vm.$refs.changePasswordTaskListTable.refresh()
						}else{
							vm.$message.error(data["message"])
						}
					}
				}).catch(function(error){})
			});
		},
		// 菜单关闭
        hideMenus() {
			this.$refs.taskMenu.hide();
        },
		// slide 提交
		submitSlide(){
			var vm = this;
			eventBus.$emit('action-save');
		},
		// 直接关闭slide事件
        hideSlide(){
			var vm = this;
			vm.$refs.changePasswordSlide.hide();
			vm.$refs.changePasswordTaskListTable.refresh();
			vm.$refs.queryResultTable.refresh();
			vm.$refs.changePasswordDeviceTable.clearSelection();
        },
		// 条件关闭slide事件 
		closeSlide(){
			var vm = this;
			eventBus.$emit('cancel-slide');
            
		},
		dateChange(val) {
			var vm = this;
			vm.dateValue = val;
			if(val != null){
				vm.queryTaskParams.startTime = vm.dateValue[0];
				vm.queryTaskParams.endTime = vm.dateValue[1];
			}
		},
	},
	mounted(){
		eventBus.$off('hide-changePassword-slide').$on('hide-changePassword-slide',this.hideSlide);
	}
	
})

</script> 
