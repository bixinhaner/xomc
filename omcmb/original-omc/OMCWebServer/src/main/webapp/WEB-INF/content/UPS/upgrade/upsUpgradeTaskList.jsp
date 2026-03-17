<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.container .cmenu{
		z-index: 361!important;
	}
	#upsUpgradeResultSlide .slide-content {
		padding: 0px!important;
	}
	.queryInfo{
		display:inline-block;
		margin-right:35px;
	}
	.queryInfo label{
		display:block;
		line-height:24px;
	}
	.exportBtn{
		width:25px;
		height:20px;
		position:absolute;
		right:35px;
		top:0px
	}
	
</style>
<div class="pageDefault" id='upsUpgradeTaskList'>
	<div class="container">
		<!-- 表格组件 -->
		<el-ctable
			:url="upsUpgradeTaskTableUrl" 
			:query-params="params" 
			ref="upsUpgradeTaskTable"
			id="upsUpgradeTaskTable"
			:height="height"
			:time="6" 
			:page-size="pageSize" 
			:page-list="pageList" 
			pagination="true">
				<!-- 列表toolbar -->
			<template slot="toolbar">
				<el-query  @query="queryUpsUpgradeFile" @advance-query="advanceQueryUpsUpgradeFile" @reset="queryReset" ok-text="<%=rb.getString("ChaXun")%>" reset-text="<%=rb.getString("ChaXunChongZhi")%>" placeholder="<%=rb.getString("RenWuMingCheng")%>">
					<template slot="form">
						<div class='queryInfo'>
							<label><%=rb.getString("RenWuMingCheng")%></label>
							<el-input v-model="queryForm.taskName"  size="mini" style='width:200px'></el-input>
						</div>
						<div class='queryInfo'>
							<label><%=rb.getString("KaiShiShiJian")%></label>
							<el-date-picker size="mini" v-model="queryForm.time"  value-format="yyyy-MM-dd HH:mm:ss" type="datetimerange" range-separator="——"  start-placeholder='<%=rb.getString("KaiShiShiJian")%>' end-placeholder='<%=rb.getString("JieShuShiJian")%>'></el-date-picker>
						</div>
					</template>
				</el-query>
			</template>
			<el-table-column prop="op" label=" " width="40" align="center">
				<template slot-scope="scope">
					<div class="el-icon el-icon-operation-more" v-clickoutside="hideMenus" @click="showMenus(scope.row,event)"></div>
				</template>
			</el-table-column>
			<el-table-column prop="TASK_NAME" label="<%=rb.getString("RenWuMingCheng")%>" min-width="250"></el-table-column>
			<el-table-column prop="CREATE_USER" label="<%=rb.getString("ChuangJianZhe")%>" min-width="150"></el-table-column>
			<el-table-column prop="CREATE_TIME" label="<%=rb.getString("ChuangJianShiJian")%>" min-width="150"></el-table-column>
			<el-table-column prop="FILE_NAME" label="<%=rb.getString("WenJianMing")%>" min-width="220"></el-table-column>
			<el-table-column prop="VERSION" label="<%=rb.getString("BanBen")%>" min-width="200"></el-table-column>
			<el-table-column prop="TASK_STATUS" label="<%=rb.getString("ZhuangTai")%>"  min-width="120">
				<template slot-scope="scope">
					<div v-html="taskTableStatus(scope.row.TASK_STATUS)"></div>
				</template>
			</el-table-column>
			<el-table-column prop="TASK_PROGRESS" label="<%=rb.getString("JinDu")%>" min-width="120"></el-table-column>
			<el-table-column prop="TASK_RESULT"  label="<%=rb.getString("JieGuo")%>" min-width="120">
				<template slot-scope="scope">
					<div v-html="taskTableResult(scope.row.TASK_RESULT)"></div>
				</template>
			</el-table-column>
			<el-table-column prop="START_TIME" label="<%=rb.getString("KaiShiShiJian")%>" min-width="140"></el-table-column>
			<el-table-column prop="END_TIME" label="<%=rb.getString("JieShuShiJian")%>" min-width="140"></el-table-column>
			
		</el-ctable>

        <!-- 菜单 -->
        <el-cmenu ref="menu" @click="menuClick" :data="menus"></el-cmenu>
		<!-- slide -->
		<el-slide ref="upsUpgradeResuitSlide" id="upsUpgradeResuitSlide" :url='slideUrl' :title="slideTitle" :footer="slideFooter" :header="slideHeader" :position="slidePosition"
			:height="slideHeight"  :width='slideWidth' @cancel="closeSlide" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
			<template slot='toolbar' v-if='exportFlag'>
				<a href='#' @click='exportUpsUpgradeTaskResult'  class='el-icon el-icon-operation-export exportBtn'></a>
			</template>
		</el-slide>
    </div>
</div>
<script type="text/javascript">
new Vue({
	el:'#upsUpgradeTaskList',
	data(){
		return {
            params:{
                timeZone:timeZone,
                searchText:'',
				taskName:'',
				startTime:'',
				endTime:''
            },
			queryForm:{
				taskName:'',
				times:''
			},
			slideUrl:'',
			slideTitle:'',
			slideHeader:'',
			slideFooter:'',
			slidePosition:'',
			slideHeight:'',
			slideWidth:'',
            height:'100%',
            pageSize:100,
			pageList:[50,100,200,500],
            menus:[],
            rowData:'',
            upsUpgradeTaskTableUrl:'${ctx}/task/upgrade/ups/getUpgradeTaskList.action',
			exportFlag:false,

		}
	},
	methods:{
		// 关闭菜单
        hideMenus() {
            this.$refs.menu.hide()
        },
		// 打开菜单
        showMenus(row,evt) {
            var vm = this;

            vm.rowData = row;
			var status = row.TASK_STATUS;

            vm.menus = [
                {label:'<%=rb.getString("JieGuo")%>',cls:"el-icon el-icon-operation-result",code:'result'},
                {label:'<%=rb.getString("KaiShi")%>',cls:"el-icon el-icon-operation-start CODE_UPS hidden",code:'start'},
                {label:'<%=rb.getString("ZanTing")%>',cls:"el-icon el-icon-operation-awaiting CODE_UPS hidden",code:'wait'},
                {label:'<%=rb.getString("ZhongZhi")%>',cls:"el-icon el-icon-operation-terminate CODE_UPS hidden",code:'end'},
                {label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
				{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_UPS hidden",code:'mod'},
                {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_UPS hidden",code:'del'},
            ];
			initTaskStatus(status,vm.menus);
            vm.$nextTick(function(){
                document.body.click();
                vm.$refs.menu.show(evt)
            })
        },
		/**
		* 菜单点击事件
		* @param ev{object}   行数据
		*/ 
        menuClick(evt) {
			var vm = this;

			var codes = {
				result:this.upsUpgradeTaskResult, // 任务结果
				start:this.upsUpgradeTaskStart,	// 开始
				wait:this.upsUpgradeTaskStop,	// 暂停
				end:this.upsUpgradeTaskTerminate,	// 终止
	    		info:this.upsUpgradeTaskInfo,  // 详情
	    		mod:this.upsUpgradeTaskModify,	// 修改
	    		del:this.upsUpgradeTaskDel,	// 删除
	    	};
			if(codes[evt.code]){
				codes[evt.code](vm.rowData["TASK_ID"],vm.rowData["TYPE"],vm.rowData["TASK_STATUS"])
			}
        },
		/**
		 * 查看执行结果
		 * @param id:当前数据id
		*/
		upsUpgradeTaskResult(id){
			var vm = this;
			vm.slideHeader = true;
			vm.slideUrl = '${ctx}/task/upgrade/ups/toTaskResult.action?task_id=' + id;
			vm.slideFooter = false;
			vm.slidePosition = 'bottom';
			vm.slideHeight = '350px';
			vm.slideWidth = '100%';
			vm.exportFlag = true;
			vm.slideTitle = '<%=rb.getString("ZhiXingJieGuo")%>';
			vm.$refs.upsUpgradeResuitSlide.showSlide(()=>{
				eventBus.$emit('ups-upgradeTaskResult-init',id);
			})
		},
		/**
		 * 激活任务
		 * @param id:数据id
		 * @param fileType: 类型
		*/
		upsUpgradeTaskStart(id,type){
			var vm = this;
			axios.post('${ctx}/task/upgrade/ups/activeTask.action',stringify({
	    		taskId : id,
	    		type:type
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.upsUpgradeTaskTable.refresh()
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
		upsUpgradeTaskStop(id,type){
			var vm = this;
			axios.post('${ctx}/task/upgrade/ups/suspendTask.action',stringify({
	    		taskId : id,
	    		type:type
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.upsUpgradeTaskTable.refresh()
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
		upsUpgradeTaskTerminate(id,type){
			var vm = this;
			axios.post('${ctx}/task/upgrade/ups/terminateUpgradeTask.action',stringify({
	    		taskId : id,
	    		type:type
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.upsUpgradeTaskTable.refresh()
	    		}else{
	    			vm.$message.error(data["message"]) //错误提示信息
	    		}
	    	}) 
		},
		/**
		 * 查看任务
		 * @param id:当前数据id
		*/
		upsUpgradeTaskInfo(id){
			var vm = this;
			eventBus.$emit('open-upsUpgrade-addTask',id,'taskView');
		},
		/**
		 * 修改任务
		 * @param id:当前数据id
		*/
		upsUpgradeTaskModify(id){
			var vm = this;
			eventBus.$emit('open-upsUpgrade-addTask',id,'taskEdit');
		},
		/**
		 * 删除升级任务
		 * @param id:当前数据id
		*/
		upsUpgradeTaskDel(id,type){
			var vm = this,
				params = {
					taskId: id,
					type:type
				};
			vm.$confirm('<%=rb.getString("QueDingShanChuWenJian")%>','<%=rb.getString("QueRen")%>').then(function(){
				axios.post('${ctx}/task/upgrade/ups/delUpgradeTask.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data) {
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});
							vm.$refs.upsUpgradeTaskTable.refresh()
						}else{
							vm.$message.error(data["message"])
						}
					}
				}).catch(function(error){})
			});
		},
		
		// 条件关闭slide事件 
		closeSlide(){
			var vm = this;
			vm.$refs.upsUpgradeResuitSlide.hide();
		},
        // 模糊搜索事件
        queryUpsUpgradeFile(val){
            var vm = this;
            vm.params.searchText = val;
        },
		// 高级查询事件
		advanceQueryUpsUpgradeFile(){
			var vm = this,
				dateValue = vm.queryForm.time;
			vm.params.taskName = vm.queryForm.taskName;
			if(dateValue != null && dateValue !== ""){
				vm.params.startTime = dateValue[0];
				vm.params.endTime = dateValue[1];
			}else{
				vm.params.startTime = '';
				vm.params.endTime = '';
			}
		},
		// 查询重置
		queryReset(){
			var vm = this;
			vm.queryForm.taskName = '';
			vm.queryForm.time = [];
		},
		// 
		 exportUpsUpgradeTaskResult(){
			var vm = this;
	    	eventBus.$emit('export-upsUpgrade-taskResult',vm.rowData.TASK_ID,vm.rowData.TYPE)
	    },
	},
	mounted(){

	}
	
})

</script> 
