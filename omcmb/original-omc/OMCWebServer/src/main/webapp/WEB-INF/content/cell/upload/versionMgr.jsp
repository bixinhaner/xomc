<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo user = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>
<%-- 版本管理界面 --%>
<style type="text/css">
.queryInfo label{
	display:block;
	margin-bottom:5px;
	line-height:26px;
}
.queryInfo{
	display:inline-block;
}
.el-form-item__error{
	margin-top:5px;
}
.fileContent .el-table__row .selected-status{
	opacity:0;
	line-height:20px;
}
.fileContent .el-table__row.current-row .selected-status,.fileContent .el-table__row:hover .selected-status{
	opacity:1;
}
.el-radio-group{
	font-size:13px;
}
.el-input__icon{
	line-height:28px;
}
.headContent{
	font-size:14px;
	color:#0F344D;
	font-weight:700;
	margin-top:30px;
	margin-bottom:20px;
}
.headContent img{
	vertical-align:top;
	margin-right:14px;
}
.addButtonDiv{
	position:absolute;
	right:20px;
	top:50px;
	width:36px;
}
.addButtonText{
	text-align:center;
	font-size:16px;
	color:#1DA3FC;
	margin-top:5px;
}
.el-date-editor .el-range-separator{
	line-height:24px;
}

.el-date-editor .el-range__icon{
	line-height:24px;
}
.el-picker-panel .el-input.el-input--small{
	width:100%;
}
.deviceItem{
	display:inline-block;
	margin-right:60px;
}
.el-input--suffix .el-input__inner{
	width:200px;
}
.timeItem .el-input--suffix .el-input__inner{
	width:220px;
}
.el-slide .el-card{
	overflow:auto;
}
</style>

<%-- 升级任务列表 --%>
<div id='taskContent' style="box-shodow: none;">
	<c:if test="${type != 'rollback' }">
		<div class="circleIcon placeholder-bt" v-if="!tabsShow" :style='styleObj' placeholder="<%=rb.getString("GuanBi")%>" @click="cancelSlide">		
			<span class="el-icon el-icon-circle-close"></span>
		</div>
		<div style='right:40px;' class="circleIcon placeholder-bt" v-if="tabsShow" :style='styleObj' placeholder="<%=rb.getString("GuanBi")%>" @click="closeTaskList">		
			<span class="el-icon el-icon-circle-close"></span>
		</div>
	</c:if>
	<c:if test="${type == 'rollback' }">
		<div class="circleIcon placeholder-bt CODE_ENB_ROLLBACK hidden" :style='styleObj' :placeholder="buttonText" @click="addRollbackTask">		
			<span class="el-icon" :class='buttonIcon'></span>
		</div>
	</c:if>
	
	<el-tabs v-model="activeName"style="height: 100%;" v-show="tabsShow">
		<!-- 软件升级 -->
		<c:if test="${type != 'rollback' }">
		<el-tab-pane label="<%=rb.getString("RuanJianShengJi")%>" name="first" class='CODE_ENB_UPGRADE_IMAGE hidden visible'>
			<table-temp :id="'enb_upgrade_list'"  :params='params_upgrade' ref='upgrade' :retain-config="true" :show-flag="true"></table-temp>
		</el-tab-pane>
		</c:if>
		<!-- 版本回退 -->
		<c:if test="${type == 'rollback' }">
		<el-tab-pane label="<%=rb.getString("ShengJiHuiTuiRenWu")%>" name="second" class='hidden-title visible'>
			<table-temp :id="'enb_rollback_list'"  :params='params_rollback' ref='rollback' :retain-config="false" :show-flag="false"></table-temp>
		</el-tab-pane>
		</c:if>
		<!-- Patch升级 -->
		<c:if test="${type != 'rollback' }">
		<el-tab-pane label="<%=rb.getString("CAZhengShuShengJi")%>" name="third" class='CODE_ENB_UPGRADE_PATCH hidden visible'>
			<table-temp :id="'enb_patch_list'"  :params='params_patch' ref='patch' :retain-config="false" :show-flag="true"></table-temp>
		</el-tab-pane>
		</c:if>
		<!-- FPGA升级 -->
		<c:if test="${type != 'rollback' }">
		<el-tab-pane label="FPGA" name="fourth" class='CODE_ENB_UPGRADE_FPGA hidden visible'>
			<table-temp :id="'enb_fpga_list'"  :params='params_fpga' ref='fpga' :retain-config="false" :show-flag="true"></table-temp>
		</el-tab-pane>
		</c:if>
	</el-tabs>
	<el-slide ref="slide" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition"
	 :height="slideHeight" :modal='modal'  :width="slideWidth" @ok='saveUpgradeTask' @cancel='cancelSlide' :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	 	<template slot='toolbar' v-if='exportFlag'>
	 		<a href='#' @click='exportMsg' style='width:25px;height:20px;position:absolute;right:45px;top:0px;' class='el-icon el-icon-operation-export'></a>
	 	</template>
	 </el-slide>
</div>
<template id='tableTemplate'>
	<div style="height: 100%;">
		<el-ctable ref="ctable" :id="tableId"  :url="url" :height="height" time=6 :query-params="params" pagination="true">
			<template slot="toolbar">
				<el-query  @query="query" @advance-query="advanceQuery" @reset="resetQuery" :placeholder="taskName" :ok-text="queryButton" :reset-text="resetButton" :arrow-text="'<%=rb.getString("GaoJiChaXun")%>'">
					<template slot="form">
						<div class='queryInfo'>
							<label><%=rb.getString("RenWuMingCheng")%></label>
							<el-input v-model="params.taskName" :value="params.taskName" size="mini" style='width:200px'></el-input>
						</div>
						<div class='queryInfo' style='margin-left:50px;'>
							<label><%=rb.getString("KaiShiShiJian")%></label>
							<el-date-picker v-model="dateValue" value-format="yyyy-MM-dd HH:mm:ss" type="datetimerange" range-separator="——"  start-placeholder='<%=rb.getString("KaiShiShiJian")%>' end-placeholder='<%=rb.getString("JieShuShiJian")%>'></el-date-picker>
						</div>
					</template>
				</el-query>
			</template>
			<el-table-column label='' width="30" class-name="no-text-tips">
				<template slot-scope="scope">
	            			<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
	          		</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("RenWuMingCheng")%>' width="400"  prop="TASK_NAME" show-overflow-tooltip="true"></el-table-column>
			<el-table-column label='<%=rb.getString("CaoZuoRen")%>' width="80" prop="CREATE_USER"></el-table-column>
			<el-table-column label='<%=rb.getString("CaoZuoShiJian")%>' width="200" prop="CREATE_TIME"></el-table-column>
			<el-table-column v-if='showFlag' label='<%=rb.getString("WenJianMing")%>' width="300" prop="FILE_NAME"></el-table-column>
			<el-table-column v-if='showFlag' label='<%=rb.getString("BanBen")%>' width="200" prop="VERSION"></el-table-column>
			<el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' width="180" prop="PRODUCT"></el-table-column>
			<el-table-column label='<%=rb.getString("ZhuangTai")%>' width="120" prop="TASK_STATUS">
				<template slot-scope="scope">
					<div v-html="taskTableStatus(scope.row.TASK_STATUS)"></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("JinDu")%>' width="100" prop="TASK_PROGRESS"></el-table-column>
			<el-table-column label='<%=rb.getString("JieGuo")%>' width="100" prop="TASK_RESULT" :formatter="resultFmt"></el-table-column>
			<el-table-column v-if="retainConfig" label='<%=rb.getString("BaoLiuPeiZhi")%>' width="200" prop="IS_KEEP_CONFIG" :formatter="configFmt"></el-table-column>
			<el-table-column label='<%=rb.getString("KaiShiShiJian")%>' width="180" prop="START_TIME"></el-table-column>
			<el-table-column label='<%=rb.getString("JieShuShiJian")%>' width="180" prop="END_TIME"></el-table-column>
		</el-ctable>
		<el-cmenu ref="menu" :data="menus" @click="clickMenu"></el-cmenu>
	</div>
</template>
<script type="text/javascript">
var enbSoftUpgradeVM = new Vue({
	el:'#taskContent',
	data:{
		activeName:'',
		params_upgrade:{
			timeZone:timeZone,
			taskName:'',
			startTime:'',
			endTime:'',
			taskType:'1',
			searchText:''
		},
		params_rollback:{
			timeZone:timeZone,
			taskName:'',
			startTime:'',
			endTime:'',
			taskType:'2',
			searchText:''
		},
		params_patch:{
			timeZone:timeZone,
			taskName:'',
			startTime:'',
			endTime:'',
			taskType:'4',
			searchText:''
		},
		params_fpga:{
			timeZone:timeZone,
			taskName:'',
			startTime:'',
			endTime:'',
			taskType:'6',
			searchText:''
		},
	    rowData:[],
	    slideUrl:'',
	    slideTitle:'',
	    slideHeader:'',
	    slideFooter:'',
	    slidePosition:'',
	    slideHeight:'',
	    slideWidth:'',
	    showTip:true,
	    modal:false,
	    taskStatus:'',
	    operateType:'',
	    addText:false,
	    exportFlag:false,
	    upgradeTab:false,
	    rollbackTab:false,
	    patchTab:false,
	    buttonIcon:'el-icon-circle-add',
	    buttonText:'<%=rb.getString("TianJia") %>',
	    ifAddFlag:true,
	    styleObj:{
	    	zIndex:10
	    },
	    tabsShow: true,
	    roolbackVisible: false
	},
	components:{
	    'table-temp':{
	    	template:'#tableTemplate',
	    	data(){
	    		return {
    				dateValue:[],
    				menus:[],
    				taskName:'<%=rb.getString("RenWuMingCheng")%>',
    				url:'${ctx}/task/upgrade/getUpgradeTaskList.action',
    				height:'100%',
    				queryButton:'<%=rb.getString("ChaXun")%>',
    				resetButton:'<%=rb.getString("ChaXunChongZhi")%>',
    				tableId: this.id
	    		}
	    	},
	    	props:['params','retainConfig','showFlag','id'],
	    	methods:{
	    		query:function(val){//软件升级模糊查询
	    			this.resetQuery();
	    			this.params.searchText  = val;
	    			this.$refs.ctable.refresh()
	    		},
	    		advanceQuery:function(){//软件升级高级查询
	    			this.params.searchText = "";
	    			if(this.dateValue != null){
	    				this.params.startTime = this.dateValue[0];
		    			this.params.endTime = this.dateValue[1];
	    			}
	    			this.$refs.ctable.refresh()
	    		},
	    		resetQuery:function(){//软件升级重置查询
	    			this.params.taskName = '';
	    			this.dateValue = []
	    		},
	    	    resultFmt(row,column,cellValue,index){//软件升级 任务结果fmt
	    	    	var resultObj = {
	    					"1" : "<%=rb.getString("ChengGong")%>",
	    					"2" : "<%=rb.getString("BuFenChengGong")%>",
	    					"3" : "<%=rb.getString("ShiBai")%>",
	    					"" : ""
	    			}
	    			return resultObj[cellValue];
	    	    },
	    	    configFmt(row,column,cellValue,index){
	    	    	if(cellValue == 'true'){
	    	    		return "<%=rb.getString("Fou")%>"
	    	    	}else{
	    	    		return "<%=rb.getString("Shi")%>"
	    	    	}
	    	    },
	    	    handerClose(){//软件升级点击页面其他地方菜单收起
	    	        this.$refs.menu.hide();
	    	    },
	    		optClick(row,ev){//软件升级点击操作出现下拉菜单  1.等待   2.进行中  3.暂停  4.已结束 5.终止中 6.暂停中
	    	    	var showCls = {
	    				'first':'CODE_ENB_UPGRADE_IMAGE hidden',
	    				'second':'CODE_ENB_ROLLBACK hidden',
	    				'third':'CODE_ENB_UPGRADE_PATCH hidden',
	    				'fourth':'CODE_ENB_UPGRADE_PATCH hidden'
	    			}
	    			var activeName = this.$root.activeName
	    	    	var status = row.TASK_STATUS;
	    	    	this.taskStatus = row.TASK_STATUS;
	    		    this.$root.rowData = row;
    		    	this.menus= [
    			          {label:'<%=rb.getString("JieGuo")%>',code:'view'},
    			          {label:'<%=rb.getString("KaiShi")%>',cls:showCls[activeName],code:'start'},
    			          {label:'<%=rb.getString("ZanTing")%>',cls:showCls[activeName],code:'stop'},
    			          {label:'<%=rb.getString("ZhongZhiRenWu")%>',cls:showCls[activeName],code:'terminate'},
    			          {label:'<%=rb.getString("XinXi")%>',code:'edit'},
    			          {label:'<%=rb.getString("XiuGai")%>',cls:showCls[activeName],code:'edit'},
    			          {label:'<%=rb.getString("ShanChu")%>',cls:showCls[activeName],code:'del'}
    			    ]
    		    	var vm = this;
    		    	initTaskStatus(status,vm.menus);
    		    	this.$nextTick(function(){
    		    		document.body.click();
	    		    	vm.$refs.menu.show(ev);
    		    	});
	    	    },
	    	    clickMenu(ev){//软件升级菜单点击方法
	    	    	var codes = {
	    	    		view:this.viewUpgradeTask,
	    	    		start:this.activeUpgradeTask,
	    	    		stop:this.suspendUpgradeTask,
	    	    		terminate:this.terminateTask,
	    	    		edit:this.editUpgradeTask,
	    	    		del:this.delUpgradeTask
	    	    	}
	    	    	if(codes[ev.code]){
	    	    		codes[ev.code](this.$root.rowData["TASK_ID"],this.$root.rowData["TYPE"],this.$root.rowData["TASK_STATUS"])
	    	    	}
	    	    },
	    	    viewUpgradeTask(task_id,type){ // 结果
	    	    	var vm = this;
	    	    	vm.$root.slideUrl = "${ctx}/task/upgrade/toUpgradeTaskProgress.action?taskId=" + task_id + "&type=" + type
	    	    	vm.$root.slideTitle = '<%=rb.getString("ZhiXingJieGuo")%>'
	    	    	vm.$root.slideFooter = false
	    	    	vm.$root.slideHeader = true
	    	    	vm.$root.slidePosition = 'bottom'
	    	    	vm.$root.slideHeight = '350px'
	    	    	vm.$root.slideWidth = '100%'
	    	    	vm.$root.operateType='view'
	    	    	vm.$root.exportFlag = true
	    	    	vm.$root.$refs.slide.showSlide(function(){
	    	    		vm.$root.modal = false;
	    	    	});
	    	    },
	    	    activeUpgradeTask(task_id,type){ // 开始
	    	    	var vm = this;
	    	    	axios.post('${ctx}/task/upgrade/activeTask.action',stringify({
	    	    		taskId : task_id,
	    	    		type : type
	    	    	})).then(function(response){
	    	    		var data = response.data;
	    	    		if(data["success"]){
	    	    			vm.$refs.ctable.refresh()
	    	    		}else{
	    	    			vm.$message.error(data["message"])
	    	    		}
	    	    	})
	    	    },
	    	    suspendUpgradeTask(task_id,type){ // 暂停
	    	    	var vm = this;
	    	    	axios.post('${ctx}/task/upgrade/suspendTask.action',stringify({
	    	    		taskId : task_id,
	    	    		type : type
	    	    	})).then(function(response){
	    	    		var data = response.data;
	    	    		if(data["success"]){
	    	    			vm.$refs.ctable.refresh()
	    	    		}else{
	    	    			vm.$message.error(data["message"])
	    	    		}
	    	    	})
	    	    },
	    	    terminateTask(task_id,type){ // 终止任务
	    	    	var vm = this;
	    	    	axios.post('${ctx}/task/upgrade/terminateUpgradeTask.action',stringify({
	    	    		taskId : task_id,
	    	    		type : type
	    	    	})).then(function(response){
	    	    		var data = response.data;
	    	    		if(data["success"]){
	    	    			vm.$refs.ctable.refresh()
	    	    		}else{
	    	    			vm.$message.error(data["message"])
	    	    		}
	    	    	})
	    	    },
	    	    editUpgradeTask(task_id,type,status){ //  信息查看
	    	    	var vm = this;
	    	    	var pane = vm.$root.activeName;
					vm.$root.modal = true;
	    	    	vm.$root.styleObj = {
	    	    			zIndex:10
	    	    	}
	    	    	vm.$root.slideHeader = true;
	    	    	if(pane == 'first'){
	    	    		vm.$root.slideUrl = '${ctx}/task/upgrade/goAddTask.action'
	    	    		if(status != 1){
	    	    			vm.$root.slideTitle = '<%=rb.getString("ChaKanShengJiRenWu")%>'
	    	    			vm.$root.slideFooter = false
	    	    		}else{
	    	    			vm.$root.slideTitle = '<%=rb.getString("XiuGaiShengJiRenWu")%>'
	    	    			vm.$root.slideFooter = true
	    	    		}
	    	    	}else if(pane == 'second'){
	    	    		vm.$root.slideUrl = '${ctx}/task/upgradeRollback/goAddTask.action'
    	    	    	if(status != 1){
    	    	    		vm.$root.slideTitle = '<%=rb.getString("XinXi")%>'
    	    	    		vm.$root.slideFooter = false
	    	    		}else{
	    	    			vm.$root.slideTitle = '<%=rb.getString("XiuGaiXiTongHuiTuiRenWu")%>'
	    	    			vm.$root.slideFooter = true
	    	    		}
	    	    	}else if(pane == 'third'){
	    	    		vm.$root.slideUrl = '${ctx}/task/upgradeCa/goAddTask.action'
    	    			if(status != 1){
    	    				vm.$root.slideTitle = '<%=rb.getString("XinXi")%>'
    	    				vm.$root.slideFooter = false
	    	    		}else{
	    	    			vm.$root.slideTitle = '<%=rb.getString("XiuGaiPatchShengJiRenWu")%>'
	    	    			vm.$root.slideFooter = true
	    	    		}
	    	    	}else if(pane == 'fourth'){
	    	    		vm.$root.slideUrl = '${ctx}/task/upgradeFpga/goAddTask.action'
	    	    			if(status != 1){
	    	    				vm.$root.slideTitle = '<%=rb.getString("XinXi")%>'
	    	    				vm.$root.slideFooter = false
		    	    		}else{
		    	    			vm.$root.slideTitle = '<%=rb.getString("XiuGaiFPGAShengJiShenWu")%>'
		    	    			vm.$root.slideFooter = true
		    	    		}
		    	    	}
	    	    	
	    	    	vm.$root.slidePosition = 'left'
	    	    	vm.$root.slideHeight = '100%'
	    	    	vm.$root.slideWidth = '90%'
	    	    	vm.$root.operateType='edit'
	    	    	vm.$root.exportFlag = false
	    			vm.$root.$refs.slide.showSlide(function(){
	    	    		eventBus.$emit('modify-task',task_id,vm.taskStatus)
	    	    	});
	    	    },
	    	    delUpgradeTask(task_id,type){//软件升级  删除任务
	    	    	var vm = this;
	    	    	this.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>',QueRen,{
	    	    		customClass:'warningConfirm',
	    	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    	    		type:'warning',
	    	    		closeOnClickModal:false
	    	    	}).then(() => {
	    	    		axios.post('${ctx}/task/upgrade/delUpgradeTask.action',stringify({
	    		    		taskId:task_id,
	    		    		type:type
	    		    	})).then(function(response){
	    		    		var data = response.data;
	    		    		if(data["success"]){
	    		    			vm.$refs.ctable.refresh()
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
	    	}
	    }
	},
	mounted(){
		this.init();
		eventBus.$on('cancel-upgrade',this.closeUpgradeTask);
		//eventBus.$on('hide-upgrade',this.hideUpgradeTask);
	},
	methods:{
		init(){
			var codes = {
					upgrade : 'first',
					ca : 'third',
					fpga : 'fourth'
			}
			if("${type}" == "rollback"){
				this.activeName = 'second';
			}else{
				this.activeName = codes[enbFileVue.activeName];
			}
		},
	    addRollbackTask(){ // 新建系统回退任务
	    	var vm = this;
	    	if(vm.ifAddFlag){
				vm.$root.slideTitle = '<%=rb.getString("XinJianXiTongHuiTuiRenWu")%>'
	    	    vm.slideUrl = '${ctx}/task/upgradeRollback/goAddTask.action'
	    	    vm.slideFooter = 'true'
	    	    vm.slideHeader = true
	    	    vm.slidePosition = 'top'
	    	    vm.slideHeight = '100%'
	    	    vm.slideWidth = '100%'
	    	    vm.operateType = 'add'
	    	    vm.exportFlag = false;
	    	    vm.$refs.slide.showSlide(function(){
	    	    	vm.modal = false
					eventBus.$emit("add-task");
	    	    });
	    	    vm.ifAddFlag = false
	    	}else{
		    	vm.cancelSlide()
	    	}
	    },
	    saveUpgradeTask(){//软件升级  保存新建任务
	    	eventBus.$emit('hander-ok');
	    },
	    cancelSlide(){//软件升级 退出新建任务页面
	    	if(this.operateType == 'view'){
	    		this.$refs.slide.hide();
	    	}else{
	    		eventBus.$emit('hander-cancel');
	    	}
	    },
	    hideUpgradeTask(){ // 关闭添加
			var vm = this;
	    	if(this.operateType == 'add'){
	    		vm.buttonIcon = 'el-icon-circle-add'
		    	vm.buttonText='<%=rb.getString("TianJia")%>'
		    	vm.ifAddFlag = true;

		    	eventBus.$emit('enb-slide-up');
	    	}
	    	vm.$refs.slide.hide();
	    },
	    closeUpgradeTask(){ // 关闭添加
			var vm = this;
	    	if(vm.operateType == 'add'){
	    		vm.buttonIcon = 'el-icon-circle-add'
		    	vm.buttonText='<%=rb.getString("TianJia")%>'
		    	vm.ifAddFlag = true
	    	}
	    	vm.$refs.slide.hide();
	    	if(vm.activeName == 'first'){
	    		vm.$refs.upgrade.$refs.ctable.refresh()
	    	}else if(vm.activeName == 'second'){
	    		vm.$refs.rollback.$refs.ctable.refresh()
	    	}else if(vm.activeName == 'third'){
	    		vm.$refs.patch.$refs.ctable.refresh()
	    	}
	    	
	    	eventBus.$emit('enb-slide-up');
	    },
		/**
		 * 导出
		 * @param text:导出文件参数及
		*/
	    exportMsg(text){
	    	eventBus.$emit('export-result',this.rowData.TASK_ID,this.rowData.TYPE)
	    },
	    closeTaskList(){
	    	enbFileVue.$refs.slide.hide();
	    }
	}
})
</script>