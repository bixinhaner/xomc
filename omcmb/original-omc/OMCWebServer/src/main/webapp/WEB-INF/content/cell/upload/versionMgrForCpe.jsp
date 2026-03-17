<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#cpeUpgrade .el-card__body{
	padding-bottom:0;
}

.el-input__icon{
	line-height:100%;
}
.transition-box .el-form-item{
	display:inline-block;
	margin-right:90px;
}
.el-picker-panel .el-input.el-input--small{
	width:100%;
}
#cpeUpgrade .operation_more{
	height:20px;
}
</style>
<div class='panelDefault' style='overflow:hidden;' id="cpeUpgrade">
	<el-topbutton :permission="'CODE_CPE_UPGRADE_IMAGE'" :icon="'el-icon-plus'" 
		:title="'<%=rb.getString("TianJia")%>'" @click="addCpeUpdadeTask" v-show="false"></el-topbutton>
	
	<el-topbutton :permission="'CODE_CPE_UPGRADE_IMAGE'" :icon="'el-icon-close'" style="z-index:2000;"
		:title="'<%=rb.getString("GuanBi")%>'" @click="eventBus.$emit('cancel-add');" v-show="!tableShow"></el-topbutton>
	<el-topbutton :permission="'CODE_CPE_UPGRADE_IMAGE visible'" :icon="'el-icon-close'" style="z-index:2000;"
		:title="'<%=rb.getString("GuanBi")%>'" @click="eventBus.$emit('enb-slide-up');" v-show="tableShow"></el-topbutton>
	
	<el-tabs v-model='activeName' style='height:100%'>
		<el-tab-pane label='<%=rb.getString("RuanJianShengJi")%>' name='first'>
			<el-ctable v-show="tableShow" :id="'cpe_upgrade_task_list'" time=6  style="padding:5px 10px;" ref="ctable" :url="url" :query-params="params" :page-size="20" :pagination=true height="100%">
				<template slot="toolbar">
					<el-query id="cqueryId" @query="query" @advance-query="advanceQuery" @reset="resetQuery" placeholder="<%=rb.getString("RenWuMingCheng")%>"
						:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
						<template slot="form">
							<el-form :model="params" ref="advanceForm" label-position="top" >
								<el-form-item label="<%=rb.getString("RenWuMingCheng")%>" prop="taskName" >
									<el-input v-model="params.taskName" size="mini"></el-input>
								</el-form-item>
								<el-form-item label="<%=rb.getString("KaiShiShiJian")%>" prop="timeRange" >
									<el-date-picker  v-model="params.timeRange" type="datetimerange" size="mini" value-format="yyyy-MM-dd HH:mm:ss"
										start-placeholder="<%=rb.getString("KaiShiShiJian")%>" end-placeholder="<%=rb.getString("JieShuShiJian")%>"></el-date-picker> 
								</el-form-item>
							</el-form>
						</template>
					</el-query>
				</template>
				<el-table-column label="" width="30" class-name="no-text-tips">
					<template slot-scope="scope"><!-- 将元素或组件表示为作用域插槽          -->
						<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose"></div>
					</template>
				</el-table-column>
				<el-table-column prop="TASK_ID"  v-if="false"></el-table-column>
				<el-table-column prop="TYPE" v-if="false"></el-table-column>
				<el-table-column prop="TASK_NAME" show-overflow-tooltip label="<%=rb.getString("RenWuMingCheng")%>"  width="450"></el-table-column>
				<el-table-column prop="CREATE_USER" label="<%=rb.getString("CaoZuoRen")%>"  width="100" ></el-table-column>
				<el-table-column prop="CREATE_TIME" label="<%=rb.getString("CaoZuoShiJian")%>" width="180"></el-table-column>
				<el-table-column prop="FILE_NAME" show-overflow-tooltip label="<%=rb.getString("WenJianMing")%>" width="200"></el-table-column>
				<el-table-column prop="VERSION" label="<%=rb.getString("BanBen")%>" width="200"></el-table-column>
				<el-table-column prop="PRODUCT" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" width="130"></el-table-column>
				<el-table-column prop="TASK_STATUS" label="<%=rb.getString("ZhuangTai")%>" width="90">
					<template slot-scope="scope">
						<div v-html="getHtml(scope.row.TASK_STATUS)"></div>
					</template>
				</el-table-column>
				<el-table-column prop="TASK_PROGRESS" label="<%=rb.getString("JinDu")%>" width="100"></el-table-column>
				<el-table-column prop="TASK_RESULT" label="<%=rb.getString("JieGuo")%>" width="80" :formatter="taskTableResult"></el-table-column>
				<el-table-column prop="START_TIME" label="<%=rb.getString("KaiShiShiJian")%>" width="180"></el-table-column>
				<el-table-column prop="END_TIME" label="<%=rb.getString("JieShuShiJian")%>" width="180"></el-table-column>
			</el-ctable>
			<el-cmenu ref="menu" :data="menus" @click="menuClick"></el-cmenu>
		</el-tab-pane>
	</el-tabs>

	<el-slide ref="slide" :url="slideUrl" :title="slideTitle" :position="position" :header="header" :footer="footer" :width="slideWidth"  :height="slideHeight"
		:modal="modal" @ok="submmit" @cancel="cancelSubmmit" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'">
		<template slot="toolbar" v-if="showExport">
			<div class="el-icon el-icon-operation-export" style='margin-right:20px;' @click="exportUpgradeProgResult"></div>
			
		</template>
	</el-slide>
</div>
<script>
var cpeSoftUpgradeVM;
$(function(){
	closeLoading();
	cpeSoftUpgradeVM = new Vue({
		el:"#cpeUpgrade",
		data(){
			return {
				url: "${ctx}/task/upgrade/cpe/getUpgradeTaskListForCpe.action",
				slideUrl:"",
				menus:[],
				position:"",
				header:"",
				footer:"",
				slideHeight:"",
				slideWidth:"",
				taskId:"",
				taskType:"",
				slideTitle:"",
				params:{
					timeZone:timeZone,
					taskName:"",
					timeRange:"",
					startTime: "",
					endTime: ""
				},
				type:'',
				modal:"",
				taskStatus:'',
				showExport:false,
				addButton:true,
				tableShow: true,
				activeName:'first'
			}
		},
    	computed:{
		},
		mounted(){
			eventBus.$off('reload').$on('reload',this.$refs.ctable.refresh);
			eventBus.$off('closeRightDiv').$on('closeRightDiv',this.cancelSubmmit);
			eventBus.$off('closeBottomDiv').$on('closeBottomDiv',this.cancelSubmmit);
		},
		watch:{
			'params.timeRange':function(){
				var value = ['',''];
				if(this.params.timeRange && this.params.timeRange.length>1) value = this.params.timeRange;
				this.params.startTime = value[0];
				this.params.endTime = value[1];
			}
		},
		methods:{
			getHtml(status) {
				return taskTableStatus(status);
			},
			addCpeUpdadeTask(){ // 新建升级任务
				this.type = 'add';
				if(this.addButton){
	        		this.addButton = false;
	            	this.slideTitle="<%=rb.getString("XinJianShengJiRenWu")%>";
					this.showExport = false;
					this.position = "top";
					this.header = true;				
					this.footer = true;
					this.slideHeight = "100%";
					this.slideUrl = "${ctx}/task/upgrade/cpe/goAddTaskForCpe.action";
					this.$refs.slide.showSlide(function(){
						this.modal = false;
						eventBus.$emit('add-task');
					});
				}else{
					eventBus.$emit('cancel-add');	
				}
			},
			modifyTask(){ // 修改升级任务
				var vm = this;
				vm.showExport = false;
				vm.type = 'modify';
				vm.slideTitle="<%=rb.getString("XiuGaiShengJiRenWu")%>";
				vm.header = true;
				vm.position = "left";
				if(vm.taskStatus == '1'){
					vm.footer = true;
					vm.slideTitle="<%=rb.getString("XiuGaiShengJiRenWu")%>";
				}else{
					vm.footer = false;
					vm.slideTitle="<%=rb.getString("ChaKanShengJiRenWu")%>";
				}
				vm.slideHeight = "100%";
				vm.slideWidth = "80%";
				vm.slideUrl = "${ctx}/task/upgrade/cpe/goAddTaskForCpe.action";
				vm.$refs.slide.showSlide(function(){
					vm.modal = true;
					eventBus.$emit('row-modify',vm.taskId,vm.taskStatus);
				});
			},
			/**
			 * 点击展开操作下拉
			 * @param row:当前的row数据
			*/
			optClick(row,event){
				var task_status = row.TASK_STATUS;
				this.taskId = row.TASK_ID;
				this.taskType = row.TYPE;
				this.taskStatus = row.TASK_STATUS;
				this.menus = [
					{label:'<%=rb.getString("JieGuo")%>',value:'0',cls:' el-icon el-icon-operation-result'},
					<%-- {label:'<%=rb.getString("KaiShi")%>',value:'1',cls:' el-icon el-icon-operation-start',show: startShow},
					{label:'<%=rb.getString("ZanTing")%>',value:'2',cls:' el-icon el-icon-operation-awaiting',show: stopShow,disable: !waitFlag},
					{label:'<%=rb.getString("ZhongZhiRenWu")%>',value:'3',cls:' el-icon el-icon-operation-terminate',show: endShow,disable: !endFlag},
					{label:buttonText,value:'4',cls:' el-icon el-icon-operation-edit'}, --%>
					{label:'<%=rb.getString("ShanChu")%>',value:'5',cls:' el-icon el-icon-operation-delete CODE_CPE_UPGRADE_IMAGE hidden'},
				];
				var vm = this;
				initTaskStatus(task_status,this.menus);
		    	this.$nextTick(function(){
		    		document.body.click();
    		    	vm.$refs.menu.show(event);
		    	});
			},
			/**
			 *导航点击
			 * @param row:传入的值
			*/
			menuClick(row, ev){
				if(row.value=="0") this.viewResult();
				if(row.value=="1" && !row.disable) this.activeTask();
				if(row.value=="2" && !row.disable) this.stopTask();
				if(row.value=="3" && !row.disable) this.terminateTask();
				if(row.value=="4") this.modifyTask();
				if(row.value=="5" && !row.disable) this.deleteTask();
			},
			// 执行结果
			viewResult(){
				this.slideTitle="<%=rb.getString("ZhiXingJieGuo")%>";
				this.type = 'view';
				this.showExport = true;
				this.position = "bottom";
				this.header = true;	
				this.footer = false;
				this.slideHeight = "300";
				this.slideWidth = "100%";
				var taskId = this.taskId;
				var taskType = this.taskType;
				this.slideUrl = '${ctx}/task/upgrade/cpe/toUpgradeTaskProgressForCpe.action?task_id=' + taskId + "&cpeType=" + taskType;
				this.$refs.slide.showSlide(function(){
					this.modal = false;
				});
			},
			// 开始
			activeTask(){
				var vm = this;
	        	axios.post('${ctx}/task/upgrade/cpe/activeTask.action',stringify(
	    				{
	    					taskId :vm.taskId,
	    					type :vm.taskType,
	    				}
	    			)).then(function(response){
	    				let data = response.data;
	    				if(data.success) vm.$refs.ctable.refresh();
	    				else
		    				vm.$alert(data["message"],'<%=rb.getString("TiShi")%>',{
								confirmButtonText:'<%=rb.getString("QueDing")%>',
								type:'error'
							}).then(function(){
			    				
			    			}).catch(function(){
			    				
			    			});
	    			}).catch(function(error){
	    				
	    			})
			},
			// 终止
			stopTask(){
				var vm=this;
	        	axios.post('${ctx}/task/upgrade/cpe/suspendTask.action',stringify(
	    				{
	    					taskId :vm.taskId,
	    					type :vm.taskType,
	    				}
	    			)).then(function(response){
	    				let data = response.data;
	    				if(data.success) vm.$refs.ctable.refresh();
	    				else
	    					vm.$alert(data["message"],'<%=rb.getString("TiShi")%>',{
								confirmButtonText:'<%=rb.getString("QueDing")%>',
								type:'error'
							}).then().catch();
	    			}).catch(function(error){
	    				
	    			})
			},
			// 修改
			terminateTask(){
				var vm=this;
	        	axios.post('${ctx}/task/upgrade/cpe/terminateUpgradeTask.action',stringify(
	    				{
	    					taskId :vm.taskId,
	    					type :vm.taskType,
	    				}
	    			)).then(function(response){
	    				let data = response.data;
	    				if(data.success) vm.$refs.ctable.refresh();
	    				else
	    					vm.$alert(data["message"],'<%=rb.getString("TiShi")%>',{
								confirmButtonText:'<%=rb.getString("QueDing")%>',
								type:'error'
							}).then().catch();
	    			}).catch(function(error){
	    				
	    			})
			},
			// 删除
			deleteTask(){
				var vm = this;
				this.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(function(){
					axios.post('${ctx}/task/upgrade/cpe/delUpgradeTask.action',stringify(
	    				{
	    					taskId :vm.taskId,
	    					type :vm.taskType,
	    				}
	    			)).then(function(response){
	    				let data = response.data;
	    				if(data.success) {
	    					vm.$refs.ctable.refresh();
	    					vm.$message({
    			    			type:'success',
    			    			message:'<%=rb.getString("ChengGong")%>'
    			    		})
	    				}
	    				else
		    				vm.$alert(data["message"],'<%=rb.getString("TiShi")%>',{
								confirmButtonText:'<%=rb.getString("QueDing")%>',
								type:'error'
							}).then().catch();
	    			}).catch(function(error){
	    				
	    			})					
				}).catch(function(){
					
				})
			},
			// 点击关闭
			handerClose(){
				this.$refs.menu.hide();
			},
			/**
			 * 输入框搜索
			 * @param val:搜索输入值
			*/
			query(val){
				this.$refs.advanceForm.resetFields();
				this.params['searchText'] = val;
				this.$refs.ctable.refresh();
			},
			// 高级查询
			advanceQuery(){
				this.params['searchText'] = "";
				this.$refs.ctable.refresh();
			},
			// 重置 查询
			resetQuery(){
				this.$refs.advanceForm.resetFields();
			},
			/**
			 * 状态转换
			 * @param cellValue:传入的 状态 数据 进行转换
			*/
			taskTableStatus(row,column,cellValue,index){
				var statusObj = {
					'1':"<%=rb.getString("DengDai")%>",	
					'2':"<%=rb.getString("JinXingZhong")%>",	
					'3':"<%=rb.getString("ZanTing")%>",	
					'4':"<%=rb.getString("YiJieShu")%>",	
					'5':"<%=rb.getString("JinXingZhong")%>",	
					'6':"<%=rb.getString("ZanTing")%>",	
					'':"",	
				}
				return statusObj[cellValue];
			},
			/**
			 * 结果转换
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
			// 导出
			exportUpgradeProgResult(){
				eventBus.$emit("export-result");
			},
			// 确定按钮
			submmit(){
				eventBus.$emit("hander-ok",this.type);
			},
			// 关闭
			cancelSubmmit(operate){
				if(operate != 'success') eventBus.$emit("hander-cancel",operate);
				else {
					this.addButton = true;
	        		  	
					this.$refs.slide.hide();
					if(this.type == 'add') eventBus.$emit('enb-slide-up');
				}
			}
		}
	})
})
</script>	