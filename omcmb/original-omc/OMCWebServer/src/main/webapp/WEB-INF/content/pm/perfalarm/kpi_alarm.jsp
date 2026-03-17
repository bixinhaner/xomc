<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>
<style>
	#kpiAlarmPage .pageBody{
		border:none;
	}
	#kpiAlarmPage .alarmSpan{
		display:inline-block;
		height:21px;
		line-height:24px;
		margin-left:8px;
	}
	#kpiAlarmPage .minor .el-icon-status-alarm:before{
		color:#D0D53B;
	}
	#kpiAlarmPage .major .el-icon-status-alarm:before{
		color:#FB9F50;
	}
	#kpiAlarmPage .critical .el-icon-status-alarm:before{
		color:#FF7B7B;
	}
	#kpiAlarmPage .warning .el-icon-status-alarm:before{
		color:#67DFF8;
	}
	.overflow-cls {
		display: inherit;
	}
</style>
<!--KPI 告警 右侧主视区内容 -->
<div class="overflow-cls">
<div class="pageDefault" id="kpiAlarmPage" style="min-width: 1200px;position: relative; border: 0; height: 100%;">
	<div class="newIconBoxCls-bt CODE_PERFORMANCE_ALARM hidden" style="right:10px;top:10px;" @click="toAddTempPage" tip="<%=rb.getString("TianJia")%>">
		<i class="el-icon el-icon-plus" ></i>
	</div>
	<!--KPI 告警表格 -->
	<div style="height: 47%;" class='commonWarp'>
		<el-ctable :id="'kpiAlarmTempList'" ref="tempList" class="pageBody" :rownumber="true" :url="tbURL" :pagination="true" :query-params="queryParams"
			@load-success="tableLoadSuccess" style='padding-top: 0px;'>
			<!--搜索工具栏 -->
			<template slot="toolbar">
				<div class='commonFlex'>
					<span class="subTitle"><%=rb.getString("KPIGaoJingMoBan")%></span>
					<div class="queryGroup commonSearchWarp">
						<el-input class='pairgrid-query' v-model="queryForm.searchText"
							placeholder="<%=rb.getString("MuBanMingCheng")%>"></el-input>
				    	<i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
					</div>
				</div>		
			</template>
			<el-table-column label='' width="74">
				<!--表格操作功能  -->
				<template slot-scope="scope">
					<div @click='resultIconClick(scope.row)' class='selectDataIcon el-icon el-icon-common-changeOperator' style='cursor: pointer; margin-right: 5px;'></div>
				
	            	<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
	          	</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("MingCheng")%>' width="200"  prop="temp_name"></el-table-column>
			<el-table-column label='<%=rb.getString("ZhuangTai")%>' width="200" prop="status">
				<template slot-scope="scope">
					<div v-if="scope.row.status == 'on'">
						<span class='el-icon el-icon-status-enable' style='margin-right:5px;'></span><span><%=rb.getString("QiYong")%></span>
					</div>
					<div v-else>
						<span class='el-icon el-icon-status-disable' style='margin-right:5px;'></span><span style='color:#CFCFCF'><%=rb.getString("JinYong")%></span>
					</div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("ZuiJinGaoJingIndex")%>' prop="index"></el-table-column>
			<el-table-column label='<%=rb.getString("ZuiJinGaoJingShiJian")%>' width="150" prop="last_time"></el-table-column>
			<el-table-column label='<%=rb.getString("YongHu")%>' width="200" prop="updator"></el-table-column>
			<el-table-column label='<%=rb.getString("GengXinShiJian")%>' prop="update_time"></el-table-column>
		</el-ctable>
	</div>
	<!-- 结果页面 -->
	<div style="height:calc(53% - 10px); margin-top: 10px;" class='commonWarp'>
		<el-ctable :id="'kpiAlarmResultsList'" ref="tempResultList" class="pageBody" :rownumber="true" :url="resultUrl" :pagination="true" :query-params="queryParams"
		 style='padding-top: 0px;'>
			<!--搜索工具栏 -->
			<template slot="toolbar">
				<div class='commonFlex'>
					<span class="subTitle"><%=rb.getString("JieGuo")%>(<span style='color: #4D84FF;'>{{resultsTemplateName}}</span>)</span>
					<div class="queryGroup commonSearchWarp" v-if='false'>
						<el-input class='pairgrid-query' v-model="queryForm.searchText"
							placeholder="<%=rb.getString("MuBanMingCheng")%>"></el-input>
				    	<i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
					</div>
				</div>	
			</template>
			<el-table-column label='<%=rb.getString("XuHao")%>' width="100"  prop="alarm_index"></el-table-column>
			<el-table-column label='<%=rb.getString("SheBeiWeiYiBiaoZhi")%>' width="180"  prop="serial_number"></el-table-column>
			<el-table-column label='<%=rb.getString("GaoJingJiBie")%>' width="100"  prop="event_type">
				<template slot-scope="scope">
					<div class="tableTdContainer minor" v-if="scope.row.event_type == 'Minor'">
						<span class="el-icon el-icon-status-alarm" style='font-size:22px;'></span><span class="alarmSpan"><%=rb.getString("CiYaoGaoJing")%></span>
					</div>
					<div class="tableTdContainer major" v-else-if="scope.row.event_type == 'Major'">
						<span class="el-icon el-icon-status-alarm" style='font-size:22px;'></span><span class="alarmSpan"><%=rb.getString("ZhuYaoGaoJing")%></span>
					</div>
					<div class="tableTdContainer critical" v-else-if="scope.row.event_type == 'Critical'">
						<span class="el-icon el-icon-status-alarm" style='font-size:22px;'></span><span class="alarmSpan" ><%=rb.getString("JinJiGaoJing")%></span>
					</div>
					<div class="tableTdContainer warning" v-else-if="scope.row.event_type == 'Warning'">
						<span class="el-icon el-icon-status-alarm" style='font-size:22px;'></span><span class="alarmSpan"><%=rb.getString("JingGaoGaoJing")%></span>
					</div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("BiaoShi")%>' width="150" prop="alarm_identifier"></el-table-column>
			<el-table-column label='<%=rb.getString("KeNengYuanYin")%>' prop="probable_cause"></el-table-column>
			<el-table-column label='<%=rb.getString("JuTiGuZhang")%>' prop="specific_problem"></el-table-column>
			<el-table-column label='<%=rb.getString("GaoJingFaShengShiJian")%>' prop="alarm_time"></el-table-column>
			<el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="status"></el-table-column>
		</el-ctable>
	</div>
	<el-cmenu ref="menu" :data="menus" @click="clickMenu"></el-cmenu>
	
	<el-slide ref="slider" :class="contClass" :url="slideUrl" :title="slideTitle" :footer="footerShow" :header='headerShow' :position="slidePosition" :modal="false" :height="sliderHeight" :width="sliderWidth" 
	  @ok='saveTemplate' @cancel='cancelSlide' :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	 	
	 </el-slide>
</div>
</div>
<script type="text/javascript">
	var alarmTemplate = new Vue({
		el: '#kpiAlarmPage',
		data(){
			
			// 变量只在当前组件生效，不影响其他组件
			return {
				tbURL: '${ctx}/pm/alarm/getKpiAlarmTempListPageData.action',
				menus:[], // 表格操作数组
				styleObj:{
			    	zIndex:10,
			    	top:12
			    },
			    queryForm: {
			    	searchText: ''
			    },
			    queryParams: {
			    	searchText: '',
			    	timeZone: timeZone
			    },
			    addText:false,
			    buttonIcon:'el-icon el-icon-circle-add',
			    buttonText:'<%=rb.getString("TianJia") %>',
			    slidePosition: 'top',
			    slideTitle: '',
			    slideUrl: '',
			    sliderHeight: '100%',
			    sliderWidth: '100%',
			    footerShow: true,
			    headerShow: true,
			    curTempId: '', // 当前点击行的id
			    contClass:'',
			    rowData:[],
			    
			    resultUrl: '',
			    resultsTemplateName: ''
			}
		},
		
		methods: {
			//模板列表加载成功回调
			tableLoadSuccess(data){
				var vm = this, id = '';
				
				if(data.rows.length > 0 ){
					id = data.rows[0].id;
					vm.resultsTemplateName = data.rows[0].temp_name;
					vm.resultUrl = '${ctx}/pm/alarm/getKpiAlarmInfoListPageData.action?tempId='+id+'&timeZone='+timeZone;
				}
			},
		
			// 搜索功能
			query(){
				var vm = this;
				Object.assign(vm.queryParams, vm.queryForm);
			},
			
			/**
			* 表格操作-下拉菜单
			* @param row{object}: 表格行数据
			* @param ev{event}: 事件
			**/
			optClick(row,ev){
    	    	var status = row.status,
    	    		//row.status == 'on' 启用，row.status == 'off' 禁用
    	    		isStopShow = status == 'on'; 
    	    	this.$root.rowData = row;
		    	this.menus= [
			          {label:'<%=rb.getString("QiYong")%>', code:'start', cls:"el-icon el-icon-operation-enable1 CODE_PERFORMANCE_ALARM",show: !isStopShow},
			          {label:'<%=rb.getString("JinYong")%>', code:'stop', cls:"el-icon el-icon-operation-disable1 CODE_PERFORMANCE_ALARM",show: isStopShow},
			          {label:'<%=rb.getString("XinXi")%>', code:'info', cls:"el-icon el-icon-operation-info"},
			          {label:'<%=rb.getString("XiuGai")%>', code:'edit', cls:"el-icon el-icon-operation-edit CODE_PERFORMANCE_ALARM"},
			          {label:'<%=rb.getString("ShanChu")%>', code:'del', cls:"el-icon el-icon-operation-delete CODE_PERFORMANCE_ALARM"}
			    ]
		    	var vm = this;
    	    	// Vue.nextTick（）， 在下次  DOM 更新循环结束后执行延迟回调，修改数据之后立即使用此方法，获取更新后的 DOM
		    	this.$nextTick(function(){
		    		document.body.click();
    		    	vm.$refs.menu.show(ev);
		    	});
    	    },
    	    
    	 	// 表格操作-菜单点击方法
    	    clickMenu(ev){
    	    	var codes = {
    	    		start: this.activeAlarmTemp, // 启用
    	    		stop: this.stopAlarmTemp, // 禁用
    	    		info: this.viewInfo, // 信息
    	    		edit: this.editAlarmTemp, // 修改
    	    		del: this.delAlarmTemp // 删除
    	    	}
    	    	if(codes[ev.code]){
    	    		codes[ev.code](this.$root.rowData["id"]);
    	    	}
    	    },
    		//点击结果图标展示逻辑
			resultIconClick(data){
				var vm = this, id = data.id;
				
				vm.resultsTemplateName = data.temp_name;
				vm.resultUrl = '${ctx}/pm/alarm/getKpiAlarmInfoListPageData.action?tempId='+id+'&timeZone='+timeZone;
			},
    
    	 	// 跳到模板查看页面
    	    viewInfo(id){
    	    	var vm = this;
    	    	
    	    	vm.contClass = 'commonBorderSlide';
    	    	vm.slideUrl = '${ctx}/pm/alarm/toKpiAlarmTempAdd.action?type=view';
    	    	vm.slideTitle = '<%=rb.getString("XinXi")%>';
    	    	vm.slidePosition = 'left';
    	    	vm.sliderHeight = '100%';
    	    	vm.sliderWidth = '75%';
    	    	vm.footerShow = false;
    	    	vm.headerShow = true;
    	    	vm.$refs.slider.showSlide(function(){
    	    		eventBus.$emit('action-init',id);
    	    	});   	    	
    	    },
    	    
    	 	// 跳到模板修改页面
    	    editAlarmTemp(id){
    	    	var vm = this;
    	    	
    	    	vm.contClass = 'commonBorderSlide';
    	    	vm.slideUrl = '${ctx}/pm/alarm/toKpiAlarmTempAdd.action?type=edit';
    	    	vm.slideTitle = '<%=rb.getString("XiuGai")%>';
    	    	vm.slidePosition = 'left';
    	    	vm.sliderHeight = '100%';
    	    	vm.sliderWidth = '75%';
    	    	vm.footerShow = true;
    	    	vm.headerShow = true;
    	    	vm.$refs.slider.showSlide(function(){
    	    		eventBus.$emit('action-init', id);
    	    	});
    	    },
    	    
    	  	// 点击页面其他地方菜单收起
			handerClose(){
    	        this.$refs.menu.hide();
    	    },
    	    
    	    showText(){
    	    	this.addText = true;
    	    },
    	    
    	    hideText(){
    	    	this.addText = false;
    	    },
    	   
    	    // 表格操作-点击启用
    	    activeAlarmTemp(id) {
    	    	var vm = this,
    	    		params = {
	    	    		tempId: id,
	    	    		status: 'on'
	    	    	};
    	    	axios.post('${ctx}/pm/alarm/updateKpiAlarmTempStatus.action',stringify(params)).then(function(response){
					var data = response.data;
					
					if(data) {
						if(data["success"]){
							vm.$message({
	    						message: '<%=rb.getString("ChengGong")%>',
	    						type:'success'
	    					});
	    					vm.$refs.tempList.refresh(); // 刷新表格
	    				}else{
	    					vm.$message.error(data["message"])
	    				}
					}
				}).catch(function(error){})
    	    },
    	    
    	    // 表格操作-点击禁用
    	    stopAlarmTemp(id) {
    	    	var vm = this,
		    		params = {
	    	    		tempId: id,
	    	    		status: 'off'
	    	    	};
		    	axios.post('${ctx}/pm/alarm/updateKpiAlarmTempStatus.action',stringify(params)).then(function(response){
					var data = response.data;
					
					if(data) {
						if(data["success"]){
							vm.$message({
	    						message: '<%=rb.getString("ChengGong")%>',
	    						type:'success'
	    					});
	    					vm.$refs.tempList.refresh();
	    				}else{
	    					vm.$message.error(data["message"]);
	    				}
					}
				}).catch(function(error){})
    	    },
    	    
    	  	// 表格操作-点击删除
    	    delAlarmTemp(id) {
    	    	var vm = this,
		    		params = {
	    	    		tempId: id
	    	    	};
    	    	vm.$confirm('<%=rb.getString("QueRenShanChuMuBan")%>','<%=rb.getString("QueRen")%>',{closeOnClickModal:false}).then(function(){
    	    		axios.post('${ctx}/pm/alarm/delKpiAlarmTemp.action',stringify(params)).then(function(response){
						var data = response.data;
						
						if(data) {
							if(data["success"]){
		    					vm.$message({
		    						message: '<%=rb.getString("ChengGong")%>',
		    						type: 'success'
		    					});
		    					vm.$refs.tempList.refresh();
		    				}else{
		    					vm.$message.error(data["message"]);
		    				}
						}
					}).catch(function(error){})
    	    	});
    	    },
    	    
    	 	// 跳到模板新增页面
    	    toAddTempPage(){
    	    	var vm = this;
    	    	
    	    	vm.contClass = 'commonBorderSlide';
    	    	vm.slideUrl = '${ctx}/pm/alarm/toKpiAlarmTempAdd.action'; // 新增  KPI 告警模板页面
    	    	vm.slideTitle = '<%=rb.getString("XinZengKPIGaoJingMoBan")%>';
    	    	vm.slidePosition = 'top';
    	    	vm.sliderHeight = '100%';
    	    	vm.footerShow = true;
    	    	vm.headerShow = true;
    	    	
    	    	// 检测模板是否达到最大数
    	    	axios.post('${ctx}/pm/alarm/checkKpiAlarmTempIsExist.action').then(function(response){
					var data = response.data;
					
					if(data) {
						if(data['success']) {
			            	vm.$root.$refs.slider.showSlide(function(){
			    	    		eventBus.$emit('action-init','');
			    	    	});
						}else {
							vm.$message.error(data["message"])
						}
					}
				}).catch(function(error){})
    	    },
    		// 触发新增或修改界面的保存操作
    	    saveTemplate(){
    	    	eventBus.$emit('action-save');
    	    },
    	    
    	 	// 关闭侧滑页
    	    cancelSlide(){
    	    	this.$refs.slider.hide();
    	    },
    	    
    	 	// 响应新增或修改界面保存成功处理
    	    saveOK(){
    	    	var vm = this;
    	    	
    	    	vm.$refs.slider.hide();
    	    	vm.$refs.tempList.refresh();
    	    }
		},
		
		// 实例被挂载后调用
		mounted(){
			closeLoading();
			eventBus.$off('action-ok').$on('action-ok',this.saveOK);
			//eventBus.$off('action-cancel').$on('action-cancel',this.cancelSlide);
		}
	});
</script>