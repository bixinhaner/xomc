<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#enbChangePassWordPage .changePassWordMainBox{
		height: 100%;
		flex: 1;
		display: flex;
		flex-direction: column;
	}
	#enbChangePassWordPage .container .cmenu{
		z-index: 361!important;
	}
	#enbChangePassWordPage .deviceTable{
		height: calc(50% - 5px);
	}
	#enbChangePassWordPage .productTypeButtonBox .el-radio-button{
		margin-right:10px;
	}
	#enbChangePassWordPage .productTypeButtonBox .el-radio-button__inner{
		border:1px solid #F6F7FB;
		background:none;
		padding:5px  15px;
	}
	#enbChangePassWordPage .productTypeButtonBox .el-radio-button--mini .el-radio-button__inner,
	#enbChangePassWordPage .productTypeButtonBox.el-radio-button:first-child .el-radio-button__inner,
	#enbChangePassWordPage .productTypeButtonBox .el-radio-button:last-child .el-radio-button__inner{
		border-radius:13px;
		border-left:1px solid #F6F7FB;
	}
	#enbChangePassWordPage .productTypeButtonBox .el-radio-button__orig-radio:checked+.el-radio-button__inner{
		color:#666;
		border-color:#4D84FF;
		background-color:#fff;
		-webkit-box-shadow:none;
	}
	#enbChangePassWordPage .flex-form {
		display: flex;
		flex-wrap: wrap;
	}
	#enbChangePassWordPage .flex-form .el-form-item {
		margin-right: 100px;
		margin-bottom: 10px;
	}
	#enbChangePassWordPage .changePassWordTaskBox{
		box-sizing: border-box;
		height: calc(50% - 5px);
		background-color: #FFFFFF;
		overflow: auto;
		margin-top: 10px;
	}

	#enbChangePassWordPage .taskHeadBox{
		position: absolute;
		top:15px;
		right: 0px;
		display: flex;
		z-index: 99;
	}
	#enbChangePassWordPage .taskHeadBox .statisticsDiv{
		height: 16px;
		line-height: 16px;
		display: flex;
		overflow: hidden;
		font-size: 14px;
	}
	#enbChangePassWordPage .statisticsDiv > div:first-child{
		padding: 0px 0 0 10px;
		color: #4D84FF;
		background: #FFFFFF;
		height: 16px;
		line-height: 16px;
	}
	#enbChangePassWordPage .statisticsDiv > div:last-child{
		padding: 0px 10px;
	}
	#enbChangePassWordPage .statisticsDiv .el-icon, #enbChangePassWordPage .statisticsSuccessDiv .el-icon, #enbChangePassWordPage .statisticsFailDiv .el-icon{
		margin-right: 6px;
		font-size: 14px;
	}
	#enbChangePassWordPage .resultHeadBox{
		position: absolute;
		top:17px;
		right: 60px;
		display: flex;
		z-index: 99;
	}
	#enbChangePassWordPage .resultHeadBox .statisticsSuccessDiv{
		height: 14px;
		line-height: 14px;
		display: flex;
		font-size: 14px;
		border-right: 1px solid #DFE2EE;
	}
	#enbChangePassWordPage .statisticsSuccessDiv .el-icon::before{
		color:#67D972;
	}
	#enbChangePassWordPage .statisticsSuccessDiv > div:first-child{
		padding: 0px 10px;
		color: #67D972;
	}
	#enbChangePassWordPage .statisticsSuccessDiv > div:last-child{
		padding: 0px 10px 0 0;
	}
	#enbChangePassWordPage .resultHeadBox .statisticsFailDiv{
		height: 14px;
		line-height: 14px;
		display: flex;
		font-size: 14px;
	}
	#enbChangePassWordPage .statisticsFailDiv .el-icon::before{
		color: #E88282;
		content:'\e6fb';
	}
	#enbChangePassWordPage .statisticsFailDiv > div:first-child{
		padding: 0px 10px;
		color: #E88282;
	}
	
	#enbChangePassWordPage .productTypeButtonBox{
		height:36px;
		width:100%;
		margin-top:10px;
		line-height:34px;
		box-sizing: border-box;
		border-top:1px solid #D5DCEC;
		border-bottom:1px solid #D5DCEC;
		margin-bottom: -10px;
	}

	#enbChangePassWordPage .el-query {
		display:inline-block;
	}
	#enbChangePassWordPage .el-date-editor .el-range__close-icon {
		line-height:20px;
	}
</style>
<!-- eNB修改密码 -->
<div class="pageDefault" id='enbChangePassWordPage' style='background:unset; border: 0;'>
	<div class="container">
		<div class="changePassWordMainBox">
			<!-- 操作按钮 -->
			<div class="operations">
				<div class="circleIcon CODE_ENB_CHANGE_PASSWORD hidden" placeholder="<%=rb.getString("XinJianRenWu")%>" @click="addTask"  style="right:20px;top:3px;">		
					<span class="el-icon el-icon-circle-add"></span>
				</div>
			</div>
		
			<div class="deviceTable commonWarp">
				<el-ctable
					:url="deviceTableUrl"
					:query-params="queryDeviceParams" 
					ref="changePasswordDeviceTable" 
					:height="height" 
					@selection-change='deviceSelect'
					:row-key="'small_cell_code'" 
					:page-size="pageSize" 
					:page-list="pageList" 
					pagination="true">
						<!-- 列表toolbar -->
					<template slot="toolbar">
						<div class='commonQuery'>
							<span class="commonText14" style='padding: 4px 0 0 10px'><%=rb.getString("ENBSheBei")%></span>
							<el-query type="normal" @query="queryDevice" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>"></el-query>
							<div class="productTypeButtonBox commonFlex" >
								<span class='commonGeneral12' style='margin-left: 10px; margin-right: 20px;'><%=rb.getString("ChangPinXingHao")%></span>
								<el-radio-group size="mini" v-model='product_type' @change="productValChange" style='margin-top: 5px;'>
									<el-radio-button v-for="item in productTypeList" :label="item.value">{{item.name}}</el-radio-button>
								</el-radio-group>
							</div>
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
					<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("XiaoZhanBianMa")%>" min-width="150"></el-table-column>
					<el-table-column prop="host_name" show-overflow-tooltip label="<%=rb.getString("HostName")%>" min-width="250" ></el-table-column>
					<el-table-column prop="software_version" show-overflow-tooltip label="<%=rb.getString("BanBen")%>"  min-width="160"></el-table-column>
					<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="190" ></el-table-column>
				
				</el-ctable>
			</div>
			<div class="changePassWordTaskBox commonWarp" >
				<el-tabs v-model="activeName" class=" newTabs" style='height: 100%;'>
					<el-tab-pane label="<%=rb.getString("RenWuLieBiao")%>" name="task">
						<div class="taskHeadBox">
							<div class="statisticsDiv">
								<div><span class="el-icon el-icon-status-waiting1"></span><%=rb.getString("DengDai")%></div>
								<div>{{statisticTaskStatus.WaitingNum}}</div>
							</div>
							<div class="statisticsDiv">
								<div><span class="el-icon el-icon-status-inProgress"></span><%=rb.getString("JinXingZhong")%></div>
								<div>{{statisticTaskStatus.ProcessingNum}}</div>
							</div>
							<div class="statisticsDiv">
								<div><span class="el-icon el-icon-status-suspend"></span><%=rb.getString("ZanTing")%></div>
								<div>{{statisticTaskStatus.SuspendedNum}}</div>
							</div>
							<div class="statisticsDiv">
								<div><span class="el-icon el-icon-status-terminate"></span><%=rb.getString("JieShu")%></div>
								<div>{{statisticTaskStatus.EndNum}}</div>
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
							<el-table-column prop="updateUser" label="<%=rb.getString("CaoZuoRen")%>"  min-width="100" ></el-table-column>
							<el-table-column prop="updateTime" label="<%=rb.getString("CaoZuoShiJian")%>" min-width="180"></el-table-column>
							<el-table-column prop="taskStatus" label="<%=rb.getString("ZhuangTai")%>" min-width="120">
								<template slot-scope="scope">
									<div v-html="changePasswordTaskTableStatus(scope.row.taskStatus)"></div>
								</template>
							</el-table-column>
							<el-table-column prop="taskProgress" label="<%=rb.getString("JinDu")%>" min-width="100"></el-table-column>
							<el-table-column prop="taskResult" label="<%=rb.getString("JieGuo")%>" min-width="100" :formatter="taskTableResult"></el-table-column>
							<el-table-column prop="startTime" label="<%=rb.getString("KaiShiShiJian")%>" min-width="180"></el-table-column>
							<el-table-column prop="endTime" label="<%=rb.getString("JieShuShiJian")%>" min-width="180"></el-table-column>
						</el-ctable>
						<el-cmenu ref="taskMenu" :data="taskMenus" @click="taskMenuClick"></el-cmenu>
					</el-tab-pane>
					<el-tab-pane label="<%=rb.getString("SheBeiLieBiao")%>" name="device"> 
						<div class="resultHeadBox">
							<div class="statisticsSuccessDiv">
								<div><span class="el-icon el-icon-circle-success"></span><%=rb.getString("ChengGong")%></div>
								<div>{{statisticDeviceResult.SucNum}}</div>
							</div>
							<div class="statisticsFailDiv">
								<div><span class="el-icon el-icon-circle-close"></span><%=rb.getString("ShiBai")%></div>
								<div>{{statisticDeviceResult.FailNum}}</div>
							</div>
						</div>
						<div class="newIconBoxCls-bt" placeholder="<%=rb.getString("DaoChu")%>" style='right: 10px; top: 10px;'>
							<span class="el-icon el-icon-circle-export" @click="exportResultTable"></span>
						</div>
						<!--:url="queryResultUrl"  :data-->
						<el-ctable 
							id="changePasswordResultTable"  class='commonQueryToolbar'
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
									<el-query type="normal" @query="queryCpeUpgradeResult" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%> / <%=rb.getString("RenWuMingCheng")%>"></el-query>
								</div>
							</template>
							<el-table-column prop="ID" v-if="false"></el-table-column>
							<el-table-column prop="serialNumber" show-overflow-tooltip label="<%=rb.getString("XiaoZhanBianMa")%>" min-width="160"></el-table-column>
							<el-table-column prop="cellName" show-overflow-tooltip label="<%=rb.getString("HostName")%>"  min-width="100"></el-table-column>
							<el-table-column prop="taskName" show-overflow-tooltip label="<%=rb.getString("RenWuMingCheng")%>"  min-width="280"></el-table-column>
							<el-table-column prop="productType" show-overflow-tooltip label="<%=rb.getString("ChangPinXingHao")%>" min-width="120"></el-table-column>
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
		<el-slide ref="changePasswordSlide" id="changePasswordSlide" :url='slideUrl' :title="slideTitle" :footer="slideFooter" :header="slideHeader" :position="slidePosition" class='commonBorderSlide'
			:height="slideHeight" :width='slideWidth' :subloading="slideSubmitLoading" @ok="submitSlide"  @cancel="closeSlide" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>
		
    </div>
</div>
<script type="text/javascript">
var changePasswordVue = new Vue({
	el:'#enbChangePassWordPage',
	data(){
		return {
			product_type:'',
			productTypeList:[],
			deviceTableUrl:"",
			productValue: "1", //产品类型
			selection:[],
			
			queryDeviceParams:{
				search_text:'',
				productValue:'',
				isUpdatePW:'true'
			},
			queryDeviceForm:{
				serial_number :'', //高级查询基站编码
				imsi:'',
				host_name :'', //高级查询基站名称
				group_id :'', //高级查询选择的设备组
				software_version:'',//高级查询选中的版本 
				model_name:'',//高级查询选中的model 
				cell_name: ''
			},
			taskListTableUrl:'${ctx}/cell/password/getCellUpdatePwdPageList.action',
			activeName:'task',
			queryTaskParams:{
				searchText:'',
				timeZone:timeZone,
				taskName:'',
				startTime:'',
				endTime:'',
			},
			dateValue: [],
			timeRange:[],
			queryTaskForm:{
				taskName:'',
				startTime:'',
				endTime:'',
			},
			queryResultUrl:'${ctx}/cell/password/getCellUpdatePwdDevicePageList.action',
			queryResultParams:{
				timeZone: timeZone,
				searchText:'',
			},
			slideUrl:'',
			slideTitle:'',
			slideHeader:true,
			slideFooter:'',
			slidePosition:'',
			slideHeight:'',
			slideWidth:'',
            slideSubmitLoading: '',

            height:'100%',
            pageSize:100,
			pageList:[50,100,200,500],
			taskMenus:[],
            rowData:'',
			taskRowData:'',
			statisticTaskStatus:{
				WaitingNum: 0,
				ProcessingNum: 0,
				SuspendedNum: 0,
				EndNum: 0
			},
			statisticDeviceResult:{
				SucNum: 0,
				FailNum: 0
			},
		}
	},
    computed:{
		isSuperAdmin() {
			return is_super_user == 'true';
		},
		hasChangeRole() {
			return writableMap.CODE_ENB_CHANGE_PASSWORD == true;
		}
    },
	watch:{
		'product_type':function(newValue,oldValue){

			var reg = new RegExp('\\\\',"g");				
			this.queryDeviceParams.productValue = newValue.replace(reg,'');
			this.deviceTableUrl = "${ctx}/task/upgrade/queryCellInfos.action";
		},
		dateValue(newVal){
			var vm = this;
			if(!newVal){
				newVal = [];
				vm.queryTaskParams.startTime = '';
				vm.queryTaskParams.endTime = '';
			}
		},
	},
	methods:{
		// 初始化
		init(){
			var vm = this;
			vm.getProductType();
		},
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
		// 新建任务
		addTask(){
			var vm = this;
			vm.slideUrl = '${ctx}/cell/password/toCellUpdatePwdTaskAdd.action';
			vm.slideFooter = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("XinJianMiMaXiuGaiRenWu")%>';
			var selectData;
			selectData = vm.productTypeList.find((item)=>{
				return  vm.product_type == item.value
			});
			vm.$refs.changePasswordSlide.showSlide(()=>{
				eventBus.$emit('action-init','','add',selectData.name);
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

			exportByForm("${ctx}/cell/password/exportCellUpdatePwdDeviceListToCSV.action",{
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
			vm.queryDeviceParams.search_text= val;
		},
		// 升级任务表格 模糊查询
		queryTask(val){
			var vm = this;

			vm.queryTaskParams.searchText= val;
		},
		dateChange(val) {
			var vm = this;
			vm.dateValue = val;
			if(val != null){
				vm.queryTaskParams.startTime = vm.dateValue[0];
				vm.queryTaskParams.endTime = vm.dateValue[1];
			}
		},
		
		// 任务表格 打开操作菜单
		taskOptClick(row,evt){
			var vm = this;

			vm.taskRowData = row;
			var status = row.taskStatus;
			vm.taskMenus = [
                {label:'<%=rb.getString("KaiShi")%>',cls:"el-icon el-icon-operation-start CODE_ENB_CHANGE_PASSWORD hidden",code:'start'},
                {label:'<%=rb.getString("ZanTing")%>',cls:"el-icon el-icon-operation-awaiting CODE_ENB_CHANGE_PASSWORD hidden",code:'wait'},
                {label:'<%=rb.getString("ZhongZhi")%>',cls:"el-icon el-icon-operation-terminate CODE_ENB_CHANGE_PASSWORD hidden",code:'end'},
				{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
				{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_ENB_CHANGE_PASSWORD hidden",code:'mod'},
                {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_ENB_CHANGE_PASSWORD hidden",code:'del'},
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
			axios.post('${ctx}/cell/password/updCellUpdatePwdTaskStatus.action',stringify({
	    		taskId: vm.taskRowData.taskId,
		    	executeType:'active'
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
			axios.post('${ctx}/cell/password/updCellUpdatePwdTaskStatus.action',stringify({
	    		taskId: vm.taskRowData.taskId,
		    	executeType:'wait'
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
			axios.post('${ctx}/cell/password/updCellUpdatePwdTaskStatus.action',stringify({
	    		taskId: vm.taskRowData.taskId,
		    	executeType:'stop'
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
			vm.slideUrl = '${ctx}/cell/password/toCellUpdatePwdTaskAdd.action?';
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
			vm.slideUrl = '${ctx}/cell/password/toCellUpdatePwdTaskAdd.action';
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
				axios.post('${ctx}/cell/password/delCellUpdatePwdTask.action',stringify(params)).then(function(response){
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
		// 获取产品类型
		getProductType(){
			var vm = this,
				params={
					isUpdatePW:"true"
				};
			
			axios.post('${ctx}/cell/version/getProductType.action',stringify(params)).then(function(response){
				vm.productTypeList = response.data;
				vm.product_type = vm.productTypeList[0].value;
				vm.queryDeviceParams.productValue = vm.productTypeList[0].value;
			}).catch(function(error){})
		},
		// 类型改变
		productValChange(){
			var vm = this;

			vm.$refs.changePasswordDeviceTable.clearSelection();
		}
	},
	mounted(){
		
		eventBus.$off('hide-changePassword-slide').$on('hide-changePassword-slide',this.hideSlide);
		this.init();
	}
	
})

</script> 
