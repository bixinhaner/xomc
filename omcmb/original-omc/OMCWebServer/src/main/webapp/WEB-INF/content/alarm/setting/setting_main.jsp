<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#alarmRulePage .pageBody {
		display:flex;
		flex-direction:column;
		flex:2;
		border: none;
		overflow: auto;
		height:100%;
	}
	
</style>
<div id='alarmRulePage' style="height:100%;">
	<!-- 按钮  添加 -->
	<div v-show="optBtnShow" class="newIconBoxCls-bt" style="right:40px;top:12px;" @click="addNewAlarmRuleTask" tip="<%=rb.getString("TianJia")%>">
		<span class="el-icon el-icon-plus"></span>
	</div>
	<template>
		<div class="pageBody">
			<el-ctable id="alarmRuleTemplateTable" :url="queryAlarmRule" :query-params="params" ref="alarmRuleTable" :height="height" :time="6" :page-size="pageSize" pagination="true">
				<template slot="toolbar">
					<div class="queryGroup" >
						<el-input placeholder="<%=rb.getString("GuiZeMingCheng")%>" v-model="searchText" @keyup.enter.native="searchResult"></el-input>
						<i class="el-icon el-icon-common-search" @click="searchResult"></i>
					</div>
				</template>
				<el-table-column prop="operation" label="" width="30">
					<template slot-scope="scope">
	            		<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
	          		</template>
				</el-table-column>
				<el-table-column prop="status" label="<%=rb.getString("ZhuangTai")%>" width="180">
					<template slot-scope="scope">
						<div class="tableTdContainer" v-if="scope.row.status == 1">
							<span class="el-icon el-icon-status-enable" style='margin-right:5px;'></span><span><%=rb.getString("QiYong")%></span>
						</div>
						<div class="tableTdContainer" v-else>
							<span class="el-icon el-icon-status-disable"></span><span style='color:#C2C2C2;margin-left:5px;'><%=rb.getString("JinYong")%></span>
						</div>
						
					</template>
				</el-table-column>
				<el-table-column prop="rule_name" label="<%=rb.getString("GuiZeMingCheng")%>"></el-table-column>
				<el-table-column prop="device_type" label="<%=rb.getString("XinGaoJingYuan")%>" :formatter="deviceTypeFormatter" ></el-table-column>
				<el-table-column prop="rule_type" :formatter="ruleTypeFormatter" label="<%=rb.getString("ZhiXingDongZuo")%>"></el-table-column>
				<el-table-column prop="user_code" label="<%=rb.getString("CaoZuoRen")%>" ></el-table-column>
				<el-table-column prop="operation_time" label="<%=rb.getString("GengXinShiJian")%>" ></el-table-column>
			</el-ctable>
			<el-cmenu ref="menu" @click="clickMenu" :data="menus"></el-cmenu>
		</div>
	</template>
</div>
<script type="text/javascript">
new Vue({
	el:'#alarmRulePage',
	data(){
		return {
			params:{
				searchText:"" // 搜索 value
			},
			searchText:'',
			rowData:[],  
			menus:[],
		    modal:false,
			queryAlarmRule:'${ctx}/cell/fault/queryAlarmRuleInfoList.action?timeZone='+timeZone,
			height:'100%',
			pageSize:50,
			loading:false,
			maxCount:'',
		}
	},
	computed:{
		optBtnShow() {
			return writableMap['CODE_ALARM_VIEW'] == true;
		},
    },
	methods:{
		// 搜索点击事件
		searchResult(){
			var vm = this;
			vm.params.searchText = vm.searchText;
		},
		// 点击页面其他地方菜单收起
	    handerClose(){
	        this.$refs.menu.hide();
	    },
		/**
		* 设备类型格式化
		* @param row{object}   行数据
		* @param column{object}   列数据
		* @param cellValue{string}   prop绑定值
		* @param index{number}   下标
		*/ 
	    deviceTypeFormatter(row,column,cellValue,index){
			if(row.default == 1){
				return 'ALL'
			}else{
				return cellValue
			}
	    	
	    },
		/**
		* 执行动作格式化
		* @param row{object}   行数据
		* @param column{object}   列数据
		* @param cellValue{string}   prop绑定值
		* @param index{number}   下标
		*/ 
	    ruleTypeFormatter(row,column,cellValue,index){
	    	var ruleType = {
	    			'0':'<%=rb.getString("JinZhiShangBao")%>',
	    			'1':'<%=rb.getString("BuRuKuBuXianShi")%>',
	    			'2':'<%=rb.getString("RuKuBuXianShi")%>',
	    			'3':'<%=rb.getString("ZiDongQueRen")%>'
	    	}
	    	return ruleType[cellValue];
	    },
		/**
		* 点击更多操作出现菜单
		* @param row{object}   行数据
		* @param ev{object}   event数据
		*/ 
	    optClick(row,ev){
	    	//在启用状态下  禁止修改 删除
			var vm = this;

	    	vm.rowData = row;
	    	var rowStatus = row.status,disableFlag = "",startFlag = "",stopFlag = "",delFlge = "";
			if(row.default == 1){
				delFlge = false;
			}else{
				delFlge = true;
			}
		    if(rowStatus == "1"){ //启用
		    	disableFlag = true;
		    	startFlag = false;
		    	stopFlag = true;
		    }else{ //停用
				
				if(row.edit == true){
					disableFlag = false;
				}else{
					disableFlag = true;
				} 
		    	startFlag = true;
		    	stopFlag = false;
		    }
	    	this.menus = [
	    		{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'view'},
	    		{label:'<%=rb.getString("QiYong")%>',cls:"el-icon el-icon-operation-enable1 CODE_ALARM_VIEW hidden",show:startFlag,code:'start'},
	    		{label:'<%=rb.getString("JinYong")%>',cls:"el-icon el-icon-operation-disable1 CODE_ALARM_VIEW hidden",show:stopFlag,code:'stop'},
	    		{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_ALARM_VIEW hidden",disable:disableFlag,code:'edit'},
	    		{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_ALARM_VIEW hidden",show:delFlge,disable:disableFlag,code:'del'},
	    	]
	    	var vm = this;
	    	this.$nextTick(function(){
	    		document.body.click();
		    	vm.$refs.menu.show(ev);
	    	})
	    },
		/**
		* 菜单点击事件
		* @param ev{object}   行数据
		*/ 
	    clickMenu(ev){
	    	var codes = {
	    		start:this.startAlarmRule,  // 启用规则
	    		view:this.viewAlarmRule,	// 查看规则
	    		stop:this.startAlarmRule,	// 停用规则
	    		edit:this.editAlarmRule,	// 修改规则
	    		del:this.deleteAlarmRule,	// 删除规则
	    	}
	    	if(ev.code == 'start' || ev.code == 'stop'){
	    		codes[ev.code](this.$root.rowData["rule_id"],this.$root.rowData["status"]);
	    	}else if(ev.code == 'level'){
	    		codes[ev.code](this.$root.rowData["rule_id"],ev.priority,this.$root.rowData["rule_name"]);
	    	}else{
	    		codes[ev.code](this.$root.rowData["rule_id"],this.$root.rowData["device_type"],this.$root.rowData["rule_name"]);
	    	}
	    },
		/**
		* 启用规则 or 禁用规则
		* @param ruleId{number}  规则id
		* @param nowstatus{number}  当前规则状态
		*/ 
	    startAlarmRule(ruleId,nowstatus){
	    	var vm = this;
	    	var status = nowstatus == 1?0:1;
			axios.post('${ctx}/cell/fault/isExistRuleInfo.action',stringify({
	    		ruleId : ruleId,
	    	})).then(function(response){
				var isExist = response.data.exist;
	    		if(isExist == 'true'){
	    			axios.post('${ctx}/cell/fault/turnAlarmRuleStatus.action',stringify({
						ruleId : ruleId,
						status:status
					})).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$refs.alarmRuleTable.refresh() //表格刷新
						}else{
							vm.$message.error(data["message"]) //错误提示信息
						}
					}) 
	    		}else{
	    			vm.$message.error('<%=rb.getString("GaoJingGuiZeBeiShanChu")%>') //错误提示信息
	    		}
	    	}) 
	    },
		/**
		* 删除过滤模板
		* @param ruleId{number}  模板id
		* @param deviceType{number}  设备类型
		* @param ruleName{string}  模板名称
		*/ 
	    deleteAlarmRule(ruleId,deviceType,ruleName){
	    	var vm = this;
	    	this.$confirm('<%=rb.getString("QueRenShanChuGaoJingGuiZe")%>','<%=rb.getString("TiShi")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		closeOnClickModal:false
	    	}).then(function(){
	    		axios.post('${ctx}/cell/fault/deleteRule.action',stringify({
		    		ruleId : ruleId,
		    		ruleName: ruleName
		    	})).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success'
						});
		    		}else{
		    			vm.$message.error(data["message"]) //错误提示信息
		    		}
					vm.$refs.alarmRuleTable.refresh() //表格刷新
		    	}) 
	    	}).catch(function(){
	    		
	    	})
	    	
	    },
		/**
		* 查看规则 
		* @param ruleId{number}  规则id
		* @param deviceType{number}  设备类型
		*/ 
	    viewAlarmRule(ruleId,deviceType){
	    	var vm = this;
			axios.post('${ctx}/cell/fault/isExistRuleInfo.action',stringify({
	    		ruleId : ruleId,
	    	})).then(function(response){
				var isExist = response.data.exist;
	    		if(isExist == 'true'){
	    			eventBus.$emit('open-filter-add','view',ruleId);
	    		}else{
	    			vm.$message.error('<%=rb.getString("GaoJingGuiZeBeiShanChu")%>') //错误提示信息
					vm.$refs.alarmRuleTable.refresh() //表格刷新
	    		}
	    	}) 
	    },
		/**
		* 修改规则 
		* @param ruleId{number}  规则id
		* @param deviceType{number}  设备类型
		*/ 
	    editAlarmRule(ruleId,deviceType){
	    	var vm = this;
			axios.post('${ctx}/cell/fault/isExistRuleInfo.action',stringify({
	    		ruleId : ruleId,
	    	})).then(function(response){
				var isExist = response.data.exist;
	    		if(isExist == 'true'){
	    			eventBus.$emit('open-filter-add','edit',ruleId);
	    		}else{
	    			vm.$message.error('<%=rb.getString("GaoJingGuiZeBeiShanChu")%>') //错误提示信息
					vm.$refs.alarmRuleTable.refresh() //表格刷新
	    		}
	    	}) 
	    },
		//点击添加按钮进入新建任务页面
	    addNewAlarmRuleTask(){
			var vm = this,maxCount;
			let sumRow = vm.$refs.alarmRuleTable.getData();
			maxCount = (parseInt(vm.maxCount)+1);
	        if(sumRow.length>= maxCount){
				var title = '<%=rb.getString("ZuiDuoCunZai")%>' +' '+ maxCount + ' '+'<%=rb.getString("ZuiDuoCunZaiEnd")%>';
	        	vm.$message.error(title,'<%=rb.getString("CuoWu")%>')
	        	return false;
	        }
			eventBus.$emit('open-filter-add','add','');
	    },
		// 新建保存loading
		loadingType(){
			var vm = this;
			vm.loading = !vm.loading
		},
		initTable(){
			this.$refs.alarmRuleTable.refresh();
		},
		getMaxCount(){
			var vm = this;
			axios.post("${ctx}/cell/fault/queryMaxAlarmRuleInfo.action").then((res) => {
				vm.maxCount = res.data.maxCount;
			})
		}

	},
	mounted(){
		this.getMaxCount();
		eventBus.$on('hide-alarmRule',this.cancelSlide);
		eventBus.$on('change-loading',this.loadingType);
		eventBus.$on('init-ruleTable',this.initTable);
	}
	
})




</script> 