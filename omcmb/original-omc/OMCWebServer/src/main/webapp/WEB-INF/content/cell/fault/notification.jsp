<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<script type="text/javascript">
	var ctx = "${ctx}";
</script>
<style type="text/css">
.viewSlide .slide-content{
	height:unset !important;
}
.viewSlide .el-ctable{
	height:100%;
}
.viewSlide .el-card .el-card__body > div:first-child{
	overflow:hidden !important;
}
</style>
<div class="pageDefault" id="notificationPage">
	<!-- 添加按钮 -->
	<div class="circleIcon CODE_ALARM_NOTIFICATION hidden" :style="zindexSty">
		<span @mouseover='showText' @mouseout='hideText' @click="addNewNotificationTask" :class='buttonIcon'></span>
		<p class='circleIconText' v-show='addText'>{{buttonText}}</p>
	</div>
	<div class="singleTitle">
		<span><%=rb.getString("AlarmEmail")%></span>
	</div>
	<!-- 表格 -->
	<template>
		<div class="pageBody">
			<el-ctable ref="notificationTable" :url="notificationTableUrl" :query-params="params" :height="height" pagination="true" :rownumber="rownumber">
				<template slot="toolbar">
					<div class="queryGroup">
						<el-input class='pairgrid-query' style='width:329px;' v-model="params.searchText" @keyup.enter.native="searchResult"
							placeholder='<%=rb.getString("MuBanMingCheng")%>' size="mini" ></el-input>
				    	<i @click='searchResult' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
					</div>
				</template>
				<el-table-column prop="operation" label="" width="30">
					<template slot-scope="scope">
	            			<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
	          		</template>
				</el-table-column>
				<el-table-column prop="temp_name" label="<%=rb.getString("MuBanMingCheng")%>" ></el-table-column>
				<el-table-column prop="state" label="<%=rb.getString("ZhuangTai")%>" width="180">
					<template slot-scope="scope">
						<div class="tableTdContainer" v-if="scope.row.state == 1">
							<span class="el-icon el-icon-status-enable" style='margin-right:5px;'></span><span><%=rb.getString("QiYong")%></span>
						</div>
						<div class="tableTdContainer" v-else>
							<span class="el-icon el-icon-status-disable"></span><span style='margin-left:5px;color:#C2C2C2'><%=rb.getString("JinYong")%></span>
						</div>
					</template>
				</el-table-column>
				<el-table-column prop="creator" label="<%=rb.getString("GengXinRen")%>" ></el-table-column>
				<el-table-column prop="modification_time" label="<%=rb.getString("GengXinShiJian")%>" ></el-table-column>
				<el-table-column prop="last_fired" label="<%=rb.getString("ZuiJinTongZhiShiJian")%>" ></el-table-column>
				<el-table-column prop="description" label="<%=rb.getString("MiaoShu")%>" ></el-table-column>
			</el-ctable>
			<el-cmenu ref="notificationMenu" @click="clickMenu" :data="menus"></el-cmenu>
		</div>
	</template>
	<el-slide ref="notificationSlide" :url='slideUrl' :title="slideTitle" :footer="slideFooter" :header="slideHeader" :position="slidePosition" v-loading="loading"
	:height="slideHeight" :modal='modal' :width='slideWidth' @ok="savaNewAlarmTask" @cancel="cancelSlide" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	</el-slide>
	<!-- 查看 slide -->
	<el-slide class="viewSlide" ref="viewSlide" :title="viewTiltle" :footer="viewFooter" :header="viewHeader" :position="viewSlidePosition" :height="viewSlideHeight" :modal='viewModal'  @cancel="cancelViewSlide">
		<el-ctable :height="height" :query-params="viewParams" ref="viewTable" :url="viewTableUrl" height="viewTableHeight" pagination="true" :rownumber="rownumber"><!--  -->
				<el-table-column prop="subject" width="280" label="<%=rb.getString("YouJianBiaoTi")%>" ></el-table-column>
				<el-table-column prop="email_address" width="280" label="<%=rb.getString("YouJianJiShouRen")%>" ></el-table-column>
				<el-table-column prop="send_time" width="190" label="<%=rb.getString("YouJianFaSongShiJian")%>" ></el-table-column>
				<%-- <el-table-column prop="state" width="90" label="<%=rb.getString("ZhuangTai")%>" >
					<template slot-scope="scope">
						<div class="tableTdContainer" v-if="scope.row.state == 1">
							<span style="margin-left:8px"><%=rb.getString("HuoYue")%></span>
						</div>
						<div class="tableTdContainer" v-else>
						<span style="margin-left:8px"><%=rb.getString("QingChu")%></span>
						</div>
					</template>
				</el-table-column> --%>
				<el-table-column prop="result" label="<%=rb.getString("JieGuo")%>" width="100">
					<template slot-scope="scope">
						<div class="tableTdContainer" v-if="scope.row.result == 0">
							<span class="tableIcon"><%=rb.getString("ShiBai")%></span>
						</div>
						<div class="tableTdContainer" v-else-if="scope.row.result == 1">
							<span class="tableIcon"><%=rb.getString("ChengGong")%></span>
						</div>
						<div class="tableTdContainer" v-else-if="scope.row.result == 2">
							<span class="tableIcon"><%=rb.getString("BuFenChengGong")%></span>
						</div>
					</template>
				</el-table-column> 
				<el-table-column prop="failure_reason" width="" label="<%=rb.getString("ShiBaiYuanYin")%>" ></el-table-column>
		</el-ctable>
	</el-slide>
</div>
<script type="text/javascript">
var eventBus = new Vue();
new Vue({
	el:"#notificationPage",
	data(){
		return{
			slideUrl:'',
			slideTitle:'',
			slideFooter:'',
			slideHeader:'',
			slidePosition:'',
			slideHeight:'',
			slideWidth:'',
			viewTableUrl:'',
			viewFooter:false,
			viewSlideHeight:'40%',
			viewTableHeight:'40%',
			viewHeader:true,
			viewSlidePosition:'bottom',
			modal:false,
			viewModal:false,
			ifAddFlag:true,
			viewTiltle:'<%=rb.getString("JieGuo")%>',
		    operateType:'',
			rownumber:true,
			viewParams:{
				temp_id:'',
				timeZone:timeZone
			},
			params:{
				searchText:'',
				timeZone:timeZone,
			},
			menus:[],
			rowData:[],
			buttonIcon:'el-icon el-icon-circle-add',
			height:'100%',
			notificationTableUrl:'${ctx}/cell/fault/queryEmailAlarmInfosPageList.action',
			addText:false,
			buttonText:'<%=rb.getString("TianJia") %>',
			pageSize:50,
			zindexSty:{
				zIndex:2000
			},
			maxNumber:5,
			loading:false
		}
	},
	methods:{
		//后台请求确认最多添加条数
		queryEamilMaxNumber(){
			var vm = this
			axios.post("${ctx}/cell/fault/queryMaxAlarmEmailInfo.action").then(function(response){
				vm.maxNumber = response.data.maxCount
			})
		},
		
		// 保存新建模板
		savaNewAlarmTask(){
			this.cancelViewSlide()
			let vm = this;
			if(!vm.ifAddFlag){
				eventBus.$emit("handle-ok-new");
			}else{
				eventBus.$emit("handle-ok-edit");
			}
			
		},
		cancelViewSlide(){
			var vm = this;
			vm.$refs.viewSlide.hide();
		},
		//取消新建模板
		cancelSlide(){
			var vm = this;
	    	eventBus.$emit('hander-cancel')
		},
		/**
		* 点击更多操作出现菜单
		* @param row{object}   行数据
		* @param ev{object}   event数据
		*/ 
		optClick(row,ev){
			//模板在启用状态 禁止修改 删除操作
			this.rowData = row;
			var rowState = row.state,
			disableFlag = "";//控制启用禁用 修改 删除 是否禁止操作
			if(rowState == '1'){ //启用状态   停用显示  修改删除 禁止操作
				disableFlag = true;
			}else{ //停用状态   启用显示  修改删除 可操作
				disableFlag = false;
			}
			//菜单的操作数据
			var vm = this;
			vm.menus = [
				{label:'<%=rb.getString("JieGuo")%>',cls:'el-icon el-icon-operation-result',code:'view'},
				{label:'<%=rb.getString("QiYong")%>',cls:'el-icon el-icon-operation-enable1 CODE_ALARM_NOTIFICATION hidden',code:'start',show:!disableFlag},
				{label:'<%=rb.getString("JinYong")%>',cls:'el-icon el-icon-operation-disable1 CODE_ALARM_NOTIFICATION hidden',code:'stop',show:disableFlag},
				{label:'<%=rb.getString("XinXi")%>',cls:'el-icon el-icon-operation-info',code:'info'},
				{label:'<%=rb.getString("XiuGai")%>',cls:'el-icon el-icon-operation-edit CODE_ALARM_NOTIFICATION hidden',code:'edit',disable:disableFlag},
				{label:'<%=rb.getString("ShanChu")%>',cls:'el-icon el-icon-operation-delete CODE_ALARM_NOTIFICATION hidden',code:'del',disable:disableFlag}
			]
			this.$nextTick(()=>{
				document.body.click();
				vm.$refs.notificationMenu.show(ev)
			})
		},
		/**
		* 菜单点击对应的方法
		* @param ev{object}   行数据
		*/ 
		clickMenu(ev){
			var codes = {
				start:this.startNotification,
				view:this.viewNotification,
				stop:this.startNotification,
				info:this.infoNotification,
				del:this.delNotification,
				edit:this.editNotification,
			};
			//根据code 判断执行哪个方法
			if(ev.code == 'start' || ev.code == 'stop'){
				codes[ev.code](this.$root.rowData['temp_id'],this.$root.rowData['state'])
			}else 
				codes[ev.code](this.$root.rowData['temp_id'])
		},
		/**
		* 启用告警通知模板 or 禁用告警通知模板
		* @param ruleId{number}  告警通知模板id
		* @param nowstatus{number}  当前模板状态
		*/ 
		startNotification(ruleId,nowstatus){
			var vm = this;
	    	var status = nowstatus == 1?0:1;
	    	axios.post('${ctx}/cell/fault/updateAlarmEmailState.action',stringify({
	    		temp_id : ruleId,
	    		state:status
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			vm.$refs.notificationTable.refresh() //表格刷新
	    		}else{
	    			vm.$message.error(data["message"]) //错误提示信息
	    		}
	    	}) 
		},
		/**
		* 查看结果页面
		* @param temp_id{number}  告警通知模板id
		*/ 
		viewNotification(temp_id){
			var vm = this;
			this.cancelViewSlide()
			vm.$refs.viewSlide.showSlide(()=>{
				vm.viewTableUrl = "${ctx}/cell/fault/queryEmailAlarmRecordsPageList.action";
				vm.viewParams.temp_id=temp_id,
				vm.$refs.viewTable.refresh()
	    	})
		},
		/**
		* 信息页面
		* @param temp_id{number}  告警通知模板id
		*/ 
		infoNotification(temp_id){
			this.cancelViewSlide()
			var vm = this;
	    	
        	vm.zindexSty = {
					zIndex:1998
			}
       	    vm.slideHeader = true;
        	vm.slideTitle = '<%=rb.getString("ChaKanMuBan")%>';
    	    vm.slideUrl = '${ctx}/cell/fault/goAlarmTempOper.action';
    	    vm.slideFooter = false;
    	    vm.slidePosition = 'right';
    	    vm.slideHeight = '100%';
    	    vm.slideWidth = '80%';
    		vm.operateType = 'view';
    		vm.$refs.notificationSlide.showSlide(()=>{
    			vm.modal = true;
    			eventBus.$emit('info-task',temp_id)
    		})
		},
		/**
		* 修改页面
		* @param temp_id{number}  告警通知模板id
		*/ 
		editNotification(temp_id){
			this.cancelViewSlide()
			var vm = this;
			vm.ifAddFlag = true;
	    	
        	vm.zindexSty = {
					zIndex:1998
			}
       	    vm.slideHeader = true;
        	vm.slideTitle = '<%=rb.getString("XiuGai")%>';
    	    vm.slideUrl = '${ctx}/cell/fault/goAlarmTempOper.action';
    	    vm.slideFooter = true;
    	    vm.slidePosition = 'right';
    	    vm.slideHeight = '100%';
    	    vm.slideWidth = '80%';
    		vm.operateType = 'view';
    		vm.$refs.notificationSlide.showSlide(()=>{
    			vm.modal = true;
    			eventBus.$emit('edit-task',temp_id)
    		})
		},
		/**
		* 删除页面
		* @param temp_id{number}  告警通知模板id
		*/ 
		delNotification(temp_id){
			var vm = this;
	    	this.$confirm('<%=rb.getString("QueRenShanChuMuBan")%>','<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		closeOnClickModal:false
	    	}).then(function(){
	    		axios.post('${ctx}/cell/fault/deleteAlarmEmailInfos.action',stringify({
		    		temp_id : temp_id,
		    	})).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
		    			vm.$refs.notificationTable.refresh() //表格刷新
		    		}else{
		    			vm.$message.error(data["message"]) //错误提示信息
		    		}
		    	}) 
	    	}).catch(function(){
	    		
	    	})
		},
		// 搜索事件
		searchResult(){
			this.$refs.notificationTable.refresh();
		},
		// 右上角添加按钮 鼠标移入
		showText(){
	    	this.addText = true;
	    },
		// 右上角添加按钮 鼠标移出
	    hideText(){
	    	this.addText = false;
	    },
	    // 新建告警通知任务
	    addNewNotificationTask(){
	    	//新建告警任务首先关闭查看信息
	    	this.cancelViewSlide();
	    	var vm = this;
	    	//判断当前是否填写 setting 中的email 判断是否可以新建
	    	axios.post("${ctx}/cell/fault/queryHasSettingEmailConfig.action").then(function(response){
				if(response.data.result == '2'){
					vm.$alert('<%=rb.getString("SheZhiYouXiangTips")%>',"<%=rb.getString("TiShi")%>",{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
					})
					return false;
				}
				
				if(response.data.result == '3'){
					vm.$alert('<%=rb.getString("MeiYouSheZhiYouXiang")%>',"<%=rb.getString("TiShi")%>",{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
					})
					return false;
				}
				
				
		    	//根据限制条数 判断能否增加
		    	let sumRow= vm.$refs.notificationTable.getData();
		    	if(sumRow.length >= vm.maxNumber){
		    		vm.$message.error("<%=rb.getString("ZuiDuoBuChaoGuoWuTiao")%>","<%=rb.getString("CuoWu")%>")
		        	return false;
		    	}
		    	if(vm.ifAddFlag){
		    		
	   	    	    vm.slideHeader = true;
	   	    	    vm.slideTitle = '<%=rb.getString("XinJianMuBan")%>'
	   	    	    vm.slideUrl = '${ctx}/cell/fault/goAlarmTempOper.action';
	   	    	    vm.slideFooter = 'false';
	   	    	    vm.slidePosition = 'top';
	   	    	    vm.slideHeight = '100%';
	   	    	    vm.slideWidth = '100%';
	   	    		vm.operateType = 'add';
	   	    		vm.$refs.notificationSlide.showSlide(()=>{
	   	    			vm.modal = false;
	   	    		})
	   	    	 	vm.ifAddFlag = false
		    	}else{
		    		<%-- vm.buttonIcon = 'el-icon-plus'
				    vm.buttonText='<%=rb.getString("TianJia")%>' --%>
				    vm.cancelSlide()
		    	}
	    	})
	    },
		// 点击页面其他地方菜单收起
	    handerClose(){
	    	this.$refs.notificationMenu.hide();
	    },
	    //点击取消导航条的操作  以及slide 的隐藏
	    hideAlarmNotification(){
	    	//如果是 信息修改页面 更改导航栏地址
	    	var vm = this;
    		vm.buttonIcon = 'el-icon el-icon-circle-add'
		    vm.buttonText='<%=rb.getString("TianJia")%>'
		    vm.ifAddFlag = true
	    	vm.$refs.notificationSlide.hide();
	    	
	    },
		// 新建页面关闭
	    closeNewAlarmTask(){
	    	var vm = this;
	    	vm.ifAddFlag = true
	    	vm.$refs.notificationSlide.hide();
	    	vm.$refs.notificationTable.refresh() 
	    },
		 // 新建保存loading
		loadingType(){
			var vm = this;
			vm.loading = !vm.loading
		}
	    
	},
	created(){
		this.queryEamilMaxNumber()
	},
	mounted(){
		eventBus.$off("hide-alarmNotification").$on("hide-alarmNotification",this.hideAlarmNotification)
		eventBus.$off("cancel-newAlarm").$on("cancel-newAlarm",this.closeNewAlarmTask)
		eventBus.$off("cancel-editAlarm").$on("cancel-editAlarm",this.closeEditAlarmTask)
		eventBus.$off("change-loading").$on('change-loading',this.loadingType);
		
	}
})
</script>