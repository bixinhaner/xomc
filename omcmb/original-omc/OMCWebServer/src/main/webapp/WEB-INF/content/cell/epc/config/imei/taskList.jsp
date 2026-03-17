<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
.device_item{
	position:relative;
}
.device_item .el-ctable-toolbar {
	padding: 0 !important;
}
.el-icon-goback:before,.el-icon-status-upgrading:before{
	color:#fff;
	font-size:14px;
}
.list_item .el-tabs__item{
	font-size:14px;
}
.list_item .el-tabs--top{
	border:none;
}
.list_item{
	position:relative;
}
.list_type{
	position:absolute;
	left:20px;
	top:7px;
	z-index: 999;
}
.task_count{
	position:absolute;
	right:10px;
	top:15px;
}
.el-badge{
	position:relative;
}
.el-badge__content{
	position:absolute;
	top:6px;
	right:-4px;
	transform:translateY(-50%) translateX(100%);
	background-color:transparent;
	border-radius:10px;
	color:#fff;
	display:inline-block;
	font-size:10px;
	height:12px;
	line-height:11px;
	padding:0 6px;
	text-align:center;
	white-space:nowrap;
	cursor:default;
	border:1px solid transparent;
}
.el-icon-star-badge:before{
	color:#F3916C;
}
.el-date-editor .el-range__close-icon{
	line-height:20px;
}
.commonTabsTop .el-tabs__header {
	border: 1px solid #D5DCEC;
	border-bottom: 0;
	border-radius: 8px 8px 0 0;
	padding: 0 20px;
}
.list_item .el-tabs__header {
	border: 0;
}
#imeiConfigPage .upgradeHeaderBoxCls{
	height: 60px;
	width: 100%;
	background: #FFF;
	display: flex;
	align-items: center;
	justify-content: center;
	margin-bottom: 10px;
}
#imeiConfigPage .upgradeHeaderBoxCls .el-icon::before{
	font-size: 16px;
}
#imeiConfigPage .upgradeHeaderBoxCls .commonRadioButton .el-radio-button__inner{
	display: flex;
	align-items: center
}
#imeiConfigPage .flex-item-cls {
	/*height: calc(100% - 70px);*/
	height: 100%;
	box-sizing: border-box;
}
#imeiConfigPage .upgradeItemBoxCls{
	height: 100%;
	position: relative;
	display: flex;
}
#imeiConfigPage .upgradeItemBoxCls >div{
	overflow: hidden;
}
#imeiConfigPage .upgradeItemBoxCls .upgradeMainPageBox{
	position: relative;
	flex: 1;
	display: flex;
}
#imeiConfigPage .upgradeItemBoxCls .importFileBoxCls{
	flex: 0 1 360px;
	margin-left: 10px;
	position: relative;
	background-color: #FFFFFF;
	box-shadow: 0px 0px 10px 1px #E9EDF9;
	border-radius: 10px;
	border: 1px solid #E9EDF9;
	box-sizing: border-box;
}
#imeiConfigPage .newTabs .el-ctable-toolbar{
	padding: 0px!important;
}
#imeiConfigPage .el-radio.is-bordered,.addDeviceDialog .el-radio.is-bordered{
	height: 30px;
	padding: 7px 20px 0 10px;
}
#imeiConfigPage .el-radio.is-bordered+.el-radio.is-bordered,.addDeviceDialog .el-radio.is-bordered+.el-radio.is-bordered{
	margin-left: 15px;
}
#imeiConfigPage .el-tabs__header{
	border-bottom: 1px solid #E9E9E9;
}
#imeiConfigPage .rightOutBoxHeadCls{
	height: 50px;
	display: flex;
	align-items: center;
	font-weight: 600;
	font-size: 14px;
	justify-content: space-between;
	padding: 0px 20px;
	border-bottom: 1px solid #E9EDF9;
}
#imeiConfigPage .rightItemMainBox{
	padding: 20px;
}
#imeiConfigPage .rightItemMainBox .el-input,#imeiConfigPage .rightItemMainBox .el-select{
	width: 100%;
}
#imeiConfigPage .importFileBoxCls .footer{
	width:100%;
	border-top:1px solid #E9E9E9;
	position:absolute;
	bottom:1px;
	height:50px;
	background:#FFFFFF;
	z-index:99;
	display: flex;
	align-items: center;
	border-radius: 0px 0px 10px 10px;
}
#imeiConfigPage .greyIcon::before{
	color: #7A7992;
	font-size: 14px;
}
#imeiConfigPage .labelSlotCls > span{
	color: #999999;
	font-size: 12px;
	margin-left: 10px;
}
#imeiConfigPage .el-radio-button:focus:not(.is-focus):not(.is-disabled){
	-webkit-box-shadow: none!important;
	box-shadow: none!important;
}
#imeiConfigPage .importFileBoxCls .el-select .el-input.is-disabled .el-input__inner,#imeiConfigPage .importFileBoxCls .el-select .el-input__inner{
    height: unset !important;
}
#imeiConfigPage .upgradeItemBoxCls .el-ctable-toolbar{
	padding: 0px!important;
}
</style>
<!-- IMEI配置任务列表 -->
<div class='panelDefault' id="imeiConfigPage" style='display:flex;flex-direction:column;'>
	<div v-if="isWritable" class="circleIcon placeholder-bt"  style="right: 100px;top:5px;" placeholder='<%=rb.getString("XinJianRenWu")%>'>
		<span class="el-icon el-icon-circle-add" @click="toIMEIConfig"></span>
	</div>
	<div class="circleIcon placeholder-bt"  style="right: 60px;top:5px;" placeholder='<%=rb.getString("WenJian")%>'>
		<span class="el-icon el-icon-circle-List" @click="toIMEIFilePage"></span>
	</div>
	<div class="circleIcon placeholder-bt"  style="right: 20px;top:5px;" placeholder='<%=rb.getString("GuanBi")%>'>
		<span class="el-icon el-icon-circle-close" @click="closeIMEIPage"></span>
	</div>
	<div class="el-card__header">
		<span>IMEI Config</span>
	</div>
	<div class="el-card__body" style="flex:1;display:flex;flex-direction:column;overflow-x:hidden;">
		<div style='height:50%;width:100%;overflow:hidden;margin-bottom:15px;' class='device_item commonTableBorder'>
			<el-ctable id="imeiConfig_table" ref="imeiConfig_table" time=6 :url="taskUrl" :height="height" :query-params="query_task_params" pagination="true" @load-success="loadSuccessTask">
				<el-table-column label='' width="30" class-name="no-text-tips">
					<template slot-scope="scope">
						<div class="el-icon el-icon-operation-more" @click="optClickTask(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
					</template>
				</el-table-column>
				<el-table-column prop='taskName' label='<%=rb.getString("RenWuMingCheng")%>'></el-table-column>
				<el-table-column prop="fileName" label='<%=rb.getString("WenJianMing")%>' width="200"></el-table-column>
				<el-table-column prop='operator' label='<%=rb.getString("CaoZuoRen")%>' width="150"></el-table-column>
				<el-table-column prop='operTime' label='<%=rb.getString("CaoZuoShiJian")%>' width="200"></el-table-column>
				<el-table-column prop="status" label='<%=rb.getString("ZhuangTai")%>' width="120" >
					<template slot-scope="scope">
						<div v-if="scope.row.status == '1'">
							<span class="el-icon el-icon-status-terminate"></span><%=rb.getString("YiJieShu")%>
						</div>
						<div v-if="scope.row.status == '2'">
							<span class="el-icon el-icon-status-inProgress"></span><%=rb.getString("JinXingZhong")%>
						</div>
						<div v-if="scope.row.status == '3'">
							<span class="el-icon el-icon-status-waiting1"></span><%=rb.getString("DengDai")%>
						</div>
						<div v-if="scope.row.status == '4'">
							<span class="el-icon el-icon-status-awaiting"></span><%=rb.getString("ZanTing")%>
						</div>
						<div v-if="scope.row.status == '5'">
							<span class="el-icon el-icon-status-terminate"></span><%=rb.getString("ZhongZhiZhong")%>
						</div>
					</template>
				</el-table-column>
				<el-table-column prop="progress" label='<%=rb.getString("JinDu")%>' width="100"></el-table-column>
				<el-table-column prop="result" label='<%=rb.getString("JieGuo")%>' width="100" :formatter="resultFmt"></el-table-column>
				
				<el-table-column prop="startTime" label='<%=rb.getString("KaiShiShiJian")%>' width="200"></el-table-column>
				<el-table-column prop="endTime" label='<%=rb.getString("JieShuShiJian")%>' width="200"></el-table-column>
				<template slot="toolbar">
					<div class='toolbarHeadBtnBoxCls commonQuery' style="height:45px;padding-left:20px;display:flex;align-items:center;">
						<span><%=rb.getString("RenWuLieBiao")%></span>

						<el-query type="normal" style='margin-left:20px;' @query="queryTask" placeholder="<%=rb.getString("RenWuMingCheng")%>"></el-query>
					</div>
					<div class='task_count' style="display: flex;" v-if="false">
						<p><span><i class='el-icon el-icon-status-waiting1'></i><%=rb.getString("DengDai")%></span><span>{{waitNum}}</span></p>
						<p><span><i class='el-icon el-icon-status-inProgress'></i><%=rb.getString("JinXingZhong")%></span><span>{{progressNum}}</span></p>
						<p><span><i class='el-icon el-icon-status-suspend'></i><%=rb.getString("ZanTing")%></span><span>{{suspendNum}}</span></p>
						<p><span><i class='el-icon el-icon-status-terminate'></i><%=rb.getString("YiJieShu")%></span><span>{{endNum}}</span></p>
					</div>
				</template>
			</el-ctable>
			<el-cmenu ref="menu_task" :data="menus_task" @click="clickMenuTask"></el-cmenu>
			
		</div>


		<div style='flex:1;width:100%;overflow:auto; border: 1px solid #D5DCEC; border-radius: 8px;' class='list_item'>					
			<el-ctable ref="imeiConfig_result_table" id="imeiConfig_result_table" time=6  :url="resultUrl" :height="height" :query-params="query_result_params" pagination="true" @load-success="loadsuccessResult">
				<el-table-column v-if="false" label='' width="30" class-name="no-text-tips">
					<template slot-scope="scope">
						<div v-if="scope.row.result == '3'" class="el-icon el-icon-operation-restart" @click="restartTask('single',scope.row.taskId,scope.row.smallCellCode)" style="cursor: pointer;"></div>
					</template>
				</el-table-column>
				<el-table-column prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>' width="200"></el-table-column>
				<el-table-column prop='hostName' label='<%=rb.getString("HostName")%>' width="300"></el-table-column>
				<el-table-column prop='taskName' label='<%=rb.getString("RenWuMingCheng")%>' width="300"></el-table-column>
				
				<el-table-column prop="status" label='<%=rb.getString("ZhuangTai")%>' width="120" >
					<template slot-scope="scope">
						<div v-if="scope.row.status == '1'">
							<span class="el-icon el-icon-status-terminate"></span><%=rb.getString("YiJieShu")%>
						</div>
						<div v-if="scope.row.status == '2'">
							<span class="el-icon el-icon-status-inProgress"></span><%=rb.getString("JinXingZhong")%>
						</div>
						<div v-if="scope.row.status == '3'">
							<span class="el-icon el-icon-status-waiting1"></span><%=rb.getString("DengDai")%>
						</div>
					</template>
				</el-table-column>
				<el-table-column prop="result" label='<%=rb.getString("JieGuo")%>' width="100" :formatter="deviceResultFmt"></el-table-column>
				<el-table-column prop='failureReason' label='<%=rb.getString("ShiBaiYuanYin")%>'></el-table-column>
				<el-table-column prop='startTime' label='<%=rb.getString("KaiShiShiJian")%>' width="200"></el-table-column>
				<el-table-column prop='endTime' label='<%=rb.getString("JieShuShiJian")%>' width="200"></el-table-column>
				<template slot="toolbar">
					
					<div class='toolbarHeadBtnBoxCls commonQuery' style='padding-left: 20px; height:45px;display:flex;align-items:center;'>
						<span><%=rb.getString("SheBeiLieBiao")%></span>
						<div class="newIconBoxCls-bt" style="right:20px;top:10px;position:absolute;" @click="exportResult" tip="<%=rb.getString("DaoChu")%>">
							<span class='el-icon el-icon-operation-export'></span>
						</div>
						<el-query type="normal" @query="queryResult" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%> / <%=rb.getString("RenWuMingCheng")%>"></el-query>
						<div style='position: absolute; right: 60px;top:15px;display:flex;'>
							<p class='suc_count'><span><i class='el-icon el-icon-circle-success'></i><%=rb.getString("ChengGong")%></span><span>{{sucNum}}</span></p>
							<p class='fail_count'><span><i class='el-icon el-icon-circle-close'></i><%=rb.getString("ShiBai")%></span><span>{{failNum}}</span></p>
						</div>
					</div>
				</template>
			</el-ctable>
		
		
		</div>
	</div>

	<el-slide ref="slide" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition" class='commonBorderSlide'
		:height="slideHeight" :modal='slideModal'  :width="slideWidth" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'"  @cancel='cancelSlide' @ok='saveSlide'>
	</el-slide>
</div>
<script>
var enbFileVue = new Vue({
	el:'#imeiConfigPage',
	data(){
		var vm = this;
		
		return{
			activeName : 'upgrade',
			height:"100%",

			file_type:'upgrade',

			slideUrl:'',
			slideTitle:'',
			slideFooter:'',
			slideHeader:'',
			slidePosition:'',
			slideHeight:'',
			slideWidth:'',
			slideModal:'',
			
			/*taskUrl:[
				{
					"id":"1",
					"taskName":"config imei",
					"fileName":"imsi imei file",
					"operator":"admin",
					"operTime":"2019-01-01 00:00:00",
					"startTime":"2019-01-01 00:00:00",
					"endTime":"2019-01-01 00:00:00",
					"status":"1",
					"result":"1",
					"progress":"3/5"
				}
			],*/
			taskUrl:'${ctx}/cell/imei/queryImsiIMEITaskPageList.action',
			dateValue:[],
			query_task_params:{
				timeZone:timeZone,
				searchText:'',
				rd: ''
			},
			
			operType:'',
			menus_task:[],
			
			rowDataTask:[],
			
			waitNum:"",
			progressNum:"",
			suspendNum:"",
			endNum:"",
			sucNum:"",
			failNum:"",

			/*resultUrl:[{
				"id":"1",
				"serialNumber":"123456",
				"cellName":"cell1",
				"taskName":"config imei",
				"status":"1",
				"result":"1",
				"failureReason":"failureReason",
				"startTime":"2019-01-01 00:00:00",
				"endTime":"2019-01-01 00:00:00"
			}],*/
			resultUrl:"${ctx}/cell/imei/queryImsiIMEITaskDetailPageList.action",
			query_result_params:{
				searchText:"",
				timeZone:timeZone,
				taskId:'',
				rd: ''
			},
			query_result_form:{
				searchText:""
			},
		
		}
	},
    computed: {
        isWritable() {
        	return writableMap.CODE_ENB_DEVICE_HALOB == true;
        },
    },
	methods:{
		init(){
			var vm = this;	
			
				
		},
		closeIMEIPage(){
			
			eventBus.$emit('hide-imei');
		},
		toIMEIConfig(){
			var vm = this;
			vm.slideHeader = true
    	    vm.slideTitle = '<%=rb.getString("XinJianRenWu")%>'
    	    vm.slideUrl = "${ctx}/cell/imei/goAddIMEITask.action"
    	    vm.slideFooter = 'true'
    	    vm.slidePosition = 'top'
    	    vm.slideHeight = '100%'
    	    vm.slideWidth = '100%'
    	    vm.operType = 'addTask'
    	    vm.$refs.slide.showSlide(function(){
    	    	vm.modal = false;
    	    });
		},
		toIMEIFilePage(){
			var vm = this;
			vm.slideHeader = true;
    	    vm.slideTitle = '<%=rb.getString("MRWenJianGuanLi")%>';
    	    vm.slideUrl = "${ctx}/cell/imei/goIMEIFile.action";
    	    vm.slideFooter = false;
    	    vm.slidePosition = 'top'
    	    vm.slideHeight = '100%'
    	    vm.slideWidth = '100%'

    	    vm.$refs.slide.showSlide(function(){
    	    	vm.modal = false;
    	    });
		},
		cancelSlide(){
		
			if(this.operType == "addTask" ||　this.operType == "modifyTask"){
				eventBus.$emit('cancel-add-task');
			}else if(this.operType == "importFile"){
				eventBus.$emit('cancel-import');
			}else if(this.operType == "editFile"){
				eventBus.$emit('cancel-edit');
			}else{
				this.$refs.slide.hide();
			}
		},
		saveSlide(){
			if(this.operType == "addTask" ||　this.operType == "modifyTask"){
				eventBus.$emit('add-task');
			}
			if(this.operType == "importFile"){
				eventBus.$emit('import-file');
			}
			if(this.operType == "editFile"){
				eventBus.$emit('edit-file');
			}
		},		
		queryTask(val){
			
				this.query_task_params.searchText = val;
				this.query_task_params.rd = Math.random();
		},
			
		resultFmt(row,column,cellValue,index){//软件升级 任务结果fmt
	    	var resultObj = {
					"1" : "<%=rb.getString("ChengGong")%>",
					"2" : "<%=rb.getString("BuFenChengGong")%>",
					"0" : "<%=rb.getString("ShiBai")%>",
					"" : ""
			}
			return resultObj[cellValue];
	    },
	    deviceResultFmt(row,column,value,index) {
	    	if (value == "0") {
				return ShiBai;
			} else if (value == "1") {
				return ChengGong;
			} else if (value == "2") {
				return ZhongZhi;
			}else {
				return "";
			}
	    },
	    
	    
	    optClickTask(row,ev){
	    	var vm = this;
   	    	var status = row.status;
   		    vm.rowDataTask = row;
			var activeDisable = true;
			if(status == '3'){
				activeDisable = false;
			}
			var stopDisable = true;
			if(status == '2'){
				stopDisable = false;
			}
			var delDisable = true;
			if(status == '1'){
				delDisable = false;
			}
			   
	    	vm.menus_task= [
		          //{label:'<%=rb.getString("KaiShi")%>',cls:'el-icon el-icon-operation-start',code:'start',disable:activeDisable},
		          {label:'<%=rb.getString("XinXi")%>',cls:'el-icon el-icon-operation-info',code:'view'},
		          {label:'<%=rb.getString("ZhongZhiRenWu")%>',cls:'el-icon el-icon-operation-terminate CODE_ENB_DEVICE_HALOB hidden',code:'terminate',disable:stopDisable},
		         // {label:'<%=rb.getString("XiuGai")%>',cls:'el-icon el-icon-operation-edit',code:'edit',disable:''},
		          {label:'<%=rb.getString("ShanChu")%>',cls:'el-icon el-icon-operation-delete CODE_ENB_DEVICE_HALOB hidden',code:'del',disable:delDisable},
		    ]
	    	
	    	vm.$nextTick(function(){
	    		document.body.click();
   		    	vm.$refs.menu_task.show(ev);
	    	});
	    },
	    clickMenuTask(ev){
	    	var codes = {
   	    		start:this.activeTask,
   	    		terminate:this.terminateTask,
				view:this.editTask,
   	    		edit:this.editTask,
   	    		del:this.delTask,
    	    }
   	    	if(codes[ev.code]){
   	    		codes[ev.code](this.rowDataTask["id"],ev.code)
   	    	}
	    },
	    activeTask(task_id){ // 开始
	    	var vm = this;
	    	axios.post('${ctx}/task/upgrade/activeTask.action',stringify({
	    		taskId : task_id,
	    		type : type
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			
	    				vm.$refs.imeiConfig_table.refresh()
	    			
	    		}else{
	    			vm.$message.error(data["message"])
	    		}
	    	})
	    },
	    terminateTask(task_id){ // 终止任务
	    	var vm = this;
	    	axios.post('${ctx}/cell/imei/updateImsiIMEITaskTerminate.action',stringify({
	    		id : task_id,
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			
	    				vm.$refs.imeiConfig_table.refresh()
	    			
	    		}else{
	    			vm.$message.error(data["message"])
	    		}
	    	})
	    },
	    editTask(task_id,code){ //  信息查看
	    	var vm = this;
	    	if(code == 'view'){
				vm.slideTitle = '<%=rb.getString("XinXi")%>';
				vm.operType = 'viewTask'
				vm.slideFooter = false
    		}else{
    			vm.slideTitle = 'Modify Task';
    			vm.operType = 'modifyTask'
    		    vm.slideFooter = true
    		}
	    	
	    	vm.slideUrl = "${ctx}/cell/imei/goAddIMEITask.action"
	    	
	    	vm.slideHeader = true
    	    vm.slidePosition = 'top'
    	    vm.slideHeight = '100%'
    	    vm.slideWidth = '100%'
    	    vm.$refs.slide.showSlide(function(){
    	    	vm.modal = false
    	    });
	    },
	    delTask(task_id){// 删除任务
	    	var vm = this;
	    	this.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>',QueRen,{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post('${ctx}/cell/imei/deleteImsiIMEITask.action',stringify({
		    		id:task_id,
		    	})).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
		    			
		    				vm.$refs.imeiConfig_table.refresh()
		    			
		    			vm.$message({
			    			type:'success',
			    			message:'<%=rb.getString("ChengGong")%>'
			    		})
		    		}else{
		    			vm.$message.error(data["message"])
		    		}
		    	}).catch(function(error){
		    		
		    	})
	    	}).catch()
	    },
		
		handerClose(){
			this.$refs.menu_task.hide();
			
		},
	
	    loadSuccessTask(data){
			if(data.properties) {
				this.waitNum = data.properties.watingCount;
				this.progressNum = data.properties.inProgressCount;
				this.suspendNum = data.properties.suspendCount;
				this.endNum = data.properties.endCount;
			}
	    },
	    loadsuccessResult(data){
			var param = {
				id:''
			}
			var vm = this;
			axios.post('${ctx}/cell/imei/getImsiIMEITaskResultStatistics.action',stringify(param)).then(function(response){
				var data = response.data;
				vm.sucNum = data.successNum;
				vm.failNum = data.failNum;
			}).catch(function(error){})
			
	    },
	    queryResult(val){
	    	
	    		Object.assign(this.query_result_params,this.query_result_form)
				this.query_result_params.searchText = val;
	    		this.query_result_params.rd = Math.random();
	    	
	    	
	    },

	    exportResult(){
	    	var vm = this;
	    	var url = "";
	    	var params = {}
	    	
	    		url = "${ctx}/cell/imei/downloadImsiIMEITaskDetailResList.action";
	    		params.timeZone = timeZone;
	    		params.searchText = vm.query_result_params.searchText;
	    	
	    	exportByForm(url,params);
	    },

	    restartTask(type,taskId,snCode){
	    	var vm = this;
	    	var result = [];
	    	if(type == 'single'){
	    		result = [{[taskId]:snCode}]
	    	}
	    	var params = {
	    			taskDeviceList : JSON.stringify(result)
	    	}
			axios.post("${ctx}/task/upgrade/reTryUpgrade.action",stringify(params)).then(function(response){
				var data = response.data;
				if(data) {
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success'
						});
						vm.$refs.imeiConfig_table.refresh()
						vm.$refs.imeiConfig_result_table.refresh()
					}else{
						vm.$message.error(data["message"])
					}
				}
			})
	    },
			
	},
	mounted(){
		this.init();
	}
})
</script>