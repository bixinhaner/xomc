<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>

<style>	
	#egwKpiMeasPage {
		height: 100%;
		display: flex;
		width: 100%;
		position: relative;
		overflow: hidden;
	}
	#egwKpiMeasPage .issueErrorColor:before {
		color: #E88282;
	}
	#egwKpiMeasPage .issueOkColor:before {
		color: #67D972;
	}
	#egwKpiMeasPage .offColor:before {
		color: #7A7992;
	}
	#egwKpiMeasPage .issueStatus {
		font-size: 20px;
	}
	#egwKpiMeasPage .kpiMeasEnableClickBox{
		height: 20px;
		width: 50px;
		position: absolute;
		top:0px;
		left: 0px;
		right: 0px;
		bottom: 0px;
		margin: auto;
		z-index: 66;
		cursor: pointer;
		opacity: 0;
	}
	#egwKpiMeasPage .el-ctable-toolbar{
		padding: 0px !important;
	}
</style>
<!-- egw 指标维护 -->
<div id="egwKpiMeasPage" class="commonWarp">
	<el-ctable 
		ref="egwKpiMeasTable" 
		:url='kpiMeasUrl' 
		:query-params="kpiMeasParams" 
		id="egwKpiMeasTable" 
		:limit="limitBatch"
		@selection-change='batchSelect'  
		:time="6" pagination="true" 
		:rownumber=true :row-key="'serialNumber'" style='width: 100%;'>
		<template slot="toolbar">
			<div class="toolbarHeadBtnBoxCls" v-show="optBtnShow">
				<div class="selectBlukBoxCls">
					<div class="selectMain">
							<div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
							<span class="el-icon-selected el-icon"></span>
							<span class="bulkSelectNumBoxCls">( {{selectionData.length}} )</span>
						</div>
						<div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="bulkSelectShow">
							<div class="selectBoxTitle">
								<span><%=rb.getString("YiXuan")%></span>
								<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable()"></span>
							</div>
							<div class="selectBoxMain">
								<div class="tableInfoCls">
									<div class="tableInfoHeader">
										<div><%=rb.getString("eGWBianMa")%></div>
										<div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
									</div>
									<el-ctable 
										id="kpiMeasBulkSelectTable" 
										ref="bulkSelectTable" 
										:data="selectionData" 
										:showHeader="false"
										:rownumber="false"
										:front-pagination="true"
										:row-key="'serialNumber'"
										height="270px" pagination="true" >
										<el-table-column prop="serialNumber" v-if="false"></el-table-column>
										<el-table-column width="588">
											<template slot-scope="scope" >
												<div class="tableItemCls">
													<span>{{scope.row.serialNumber}}</span>
													<span @click="delBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
												</div>
											</template>
										</el-table-column>
									</el-ctable>
								</div>
							</div>
						</div>
					</div>
				</div>
				<div :class="selectionData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchKpiMeasEnable('on')">
					<span class="el-icon el-icon-KPI-Meas"></span>
					<span><%=rb.getString("QiYong")%></span>
				</div>
				<div :class="selectionData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchKpiMeasEnable('off')">
					<span class="el-icon el-icon-operation-CancelMeasure"></span>
					<span><%=rb.getString("JinYong")%></span>
				</div>
			</div>	
			<div style="position:relative;">
				<div id="tableHeadQuery" class="tableHeadQueryBoxCls">
					<div class="headQueryBox">
						<div class="queryGroup">
							<el-input v-model="searchText" @keyup.enter.native="query" :placeholder='placeholderText' style="width:260px;"></el-input>
							<i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
						</div>
					</div>
					<div v-for="(item,index) in advancedQueryItemList">
						<div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
							<el-popfilter
								:label='item.label'
								v-model="item.checkedItemList"
								:list="item.options"
								:visible.sync="item.isShow"
								@check-change="advanceQuery(item.type,item.value,item.checkedItemList)">
							</el-popfilter>
						</div>
						<div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
							<el-popfilter
								type="single"
								:label='item.label'
								v-model="item.selectVal"
								:list="item.options"
								:visible.sync="item.isShow"
								@check-change="advanceQuery(item.type,item.value,item.selectVal)">
							</el-popfilter>
						</div>
					</div>
					<div class="advancedQueryItemBox"  style="background: #FFF;margin-right:10px;" @click="clearFilterClick">
						<%=rb.getString("QingKongShaiXuan")%>
					</div>
				</div>
			</div>	
        </template>
		<el-table-column label='' width="50" type="selection" :reserve-selection="true" v-if="optBtnShow"></el-table-column>
		<el-table-column label='' width="60" prop="" align="center">
			<template slot-scope="scope">
				<div class="el-icon el-icon-operation-result" title='<%=rb.getString("WenJian") %>' @click="openKpiMeasFilePage(scope.row)"></div>
			</template>
		</el-table-column>
        <el-table-column label="<%=rb.getString("ShiFouQiYong")%>" prop="measure_enable" width="90" v-if="optBtnShow">
			<template slot-scope="scope">
				<el-switch :ref="scope.row.serialNumber" v-model="scope.row.measure_enable" style="height: 18px;margin-left: 10px;"
					active-color="#4D84FF"
					active-value="ON"
					inactive-value="OFF" 
					:disabled="writableMap['CODE_EGW'] == false">
				</el-switch>
				<div class="kpiMeasEnableClickBox" @click="switchChange(scope.row)"></div>
			</template>
		</el-table-column>
		<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="status" sortable>
        	<template slot-scope="scope">				
				<div v-if="scope.row.status == '0'" class="statusTip">	
					<span class="el-icon el-icon-status-kpi-off issueStatus offColor"></span>
					<%=rb.getString("Guan")%> 
				</div>
				<div v-else-if="scope.row.status == '1'" class="statusTip">
					<span class="el-icon el-icon-status-kpi-normal issueStatus issueOkColor"></span>
					<%=rb.getString("ZhengChang")%> 
				</div>
				<div v-else-if="scope.row.status == '2'" class="statusProgress">			
					<span class="el-icon el-icon-status-kpi-failure issueStatus issueErrorColor"></span>
					 <%=rb.getString("SuiHuaiZhongZhi")%>  	
				</div>				
			</template>
        </el-table-column>      
		<el-table-column label="<%=rb.getString("eGWBianMa")%>" prop="serialNumber" sortable></el-table-column> 
        <el-table-column label="<%=rb.getString("EGWMingCheng")%>" prop="gwName" sortable></el-table-column>         
        <el-table-column label="<%=rb.getString("CeLiangZhouQi")%>(<%=rb.getString("FenZhongDaXie")%>)" prop="reportPeriod"></el-table-column>    
        <el-table-column label="<%=rb.getString("KaiShiShiJian")%>" prop="startTime" sortable></el-table-column>                        
        <el-table-column label="<%=rb.getString("GengXinShiJian")%>" prop="updateTime" sortable></el-table-column>                        
	</el-ctable>	

	 <!-- 二级页面 -- 测量维护文件页面 -->
	 <el-slide ref="slideView" :url="slideUrl" :title="slideTitleView" :footer="false" :header='slideHeader' :position="slidePosition" :force-position="true"
	    :height="slideHeight" :modal='modal' :width="slideWidth" @cancel='cancelViewSlide'>
	</el-slide>
</div>

<script type="text/javascript">
	var egwKpiMeasPageVue = new Vue({
	    el: '#egwKpiMeasPage',
	    data() {
	    	return {
	    		kpiMeasParams: {
	    			timeZone: timeZone,
	             	searchText: '',
					status:'',
					measEnable:'',
	            }, 
	            kpiMeasUrl: '${ctx}/egw/pm/customizemg/getCustomizeListPageData.action',
				selectionData:[],
				bulkSelectShow:false,
				searchText:'',
				placeholderText:'<%=rb.getString("eGWBianMa")%>/<%=rb.getString("eGWMingCheng")%>',
				advancedQueryItemList:[
					{
						type:'select',
						isShow:true,
						popoverShow:false,
						selectVal:'',
						label:'<%=rb.getString("ZhuangTai") %>',
						options:[
							{value:'',label:'<%=rb.getString("QuanBu")%>'},
							{value:'0',label:'<%=rb.getString("Guan")%>'},
							{value:'1',label:'<%=rb.getString("ZhengChang")%>'},
							{value:'2',label:'<%=rb.getString("SuiHuaiZhongZhi")%>'}
						],
						value:'status',
					},
					{
						type:'select',
						isShow:true,
						popoverShow:false,
						selectVal:'',
						label:'<%=rb.getString("SheZhiKaiGuan") %>',
						options:[
							{value:'',label:'<%=rb.getString("QuanBu")%>'},
							{value:'ON',label:'<%=rb.getString("QiYong")%>'},
							{value:'OFF',label:'<%=rb.getString("JinYong")%>'}
						],
						value:'measEnable',
					},
				],
				slideUrl:'',
				slideTitle:'',
				slideTitleView:'',
				slideHeader:'',
				slideFooter:'',
				slidePosition:'',
				slideHeight:'',
				slideWidth:'',
				height:'100%',
				width:'100%',
				modal:false,
	    	}
	    },
		computed: {
			limitBatch(){
				return batchOperation ? '' : 1;
			},
			optBtnShow() {
				return writableMap['CODE_PERFORMANCE_MEASUREMENT'] == true;
			},
		},
	    methods: {
			// 模糊搜索
			query(){
				var vm = this;
				vm.kpiMeasParams.searchText = vm.searchText;
				
			},
			// 高级查询 确定事件
			advanceQuery(type,paramsItem,value){
				var vm = this,
					params ={};
				if(type == 'select'){
					params[paramsItem] = value;
				}else{
					params[paramsItem] = value.join(',');
				}
				Object.assign(vm.kpiMeasParams, params);
			},
			// 清除筛选
			clearFilterClick(){
				var vm = this,
					type = vm.activeName,
					params = {};
				vm.advancedQueryItemList.map((items)=>{
					if(items.isShow && items.isShow== true ){
						if(items.type == 'checkbox'){
							items.checkedItemList = [];
							items.oldCheckedItemList = [];
							items.checkAll = false;
							items.isIndeterminate = false;
						}else if(items.type == 'select'){
							items.selectVal = '';
						}
						if(items.type == 'checkbox' || items.type == 'select'){
							params[items.value] = '';
						}
					}
				})
				Object.assign(vm.kpiMeasParams, params);
				document.body.click();
			},
	    	//// 测量维护任务开关事件  status 0-off 1-normal 2-broken
			switchChange(row) {
				var vm = this, 
					params = {}, 
					url = '${ctx}/egw/pm/customizemg/setSingleEgwReportKPISwitch.action',
					confirmMsg = '';
				
				if((row.measure_enable == 'OFF' || row.measure_enable === undefined || row.measure_enable === null) && row.status == '0'){
					params.activeReport= 'ON';
					//egw 无重启
					confirmMsg = "<%=rb.getString("QueDingKaiQiCeLiangRenWu")%>"; 
				}else{
					params.activeReport= 'OFF';
					confirmMsg = "<%=rb.getString("QueRenGuanBiCeLiangRenWu")%>"; 
				}
				
				params.egwCodes = row.egwCode;
				vm.$confirm(confirmMsg,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post(url,stringify(params)).then(function(response){
						let data = response.data
						if(data.success){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success',
							});
							vm.$refs["egwKpiMeasTable"].clearSelection();
							vm.$refs['egwKpiMeasTable'].refresh();
						}else{
							vm.$message({
								message: data.message,
								type:'error',
							});
						}
					}).catch(function(error){})
				}).catch()
			},
			/**
			* 多选响应
			* @param selection:选择的数据
			*/
			batchSelect(selection){
				var vm = this;

				vm.selectionData = selection
			},
			// 批量操作 ———— 测量维护任务 开关
			batchKpiMeasEnable(status){
				var vm = this,
					codeList=[],
					confirmMsg='',
					isReboot = false,
					url='${ctx}/egw/pm/customizemg/setSingleEgwReportKPISwitch.action',
					params = {
						activeReport: status == 'off'? 'OFF' : 'ON',
						egwCodes:'',
					};
				if(vm.selectionData.length <= 0)return
				vm.selectionData.map(function(item,idx){
					codeList.push(item.egwCode);
				});
				params.egwCodes = codeList.join(',');
				if(status === 'on'){
					confirmMsg = '<%=rb.getString("QueDingKaiQiCeLiangRenWu")%>'
				}else{
					confirmMsg = '<%=rb.getString("QueRenGuanBiCeLiangRenWu")%>'
				}
				vm.$confirm(confirmMsg,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post(url,stringify(params)).then(function(response){
						let data = response.data
						if(data.success){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success',
							});
							vm.$refs["egwKpiMeasTable"].clearSelection();
							vm.$refs['egwKpiMeasTable'].refresh();
						}else{
							vm.$message({
								message: data.message,
								type:'error',
							});
						}
					}).catch(function(error){})
				}).catch()
			},
			// 打开已选弹窗
			openBulkSelectTable(type){
				var vm = this;
				vm.bulkSelectShow = true
			},
			// 关闭已选弹窗
			closeBulkSelectTable(type){
				var vm = this;
				vm.bulkSelectShow = false;
			},
			// 设备已选表格 清空事件
			clearBulkSelected(){
				var vm = this;
				vm.$refs["egwKpiMeasTable"].clearSelection();
				
			},
			// 设备已选表格 单个删除事件
			delBulkSelected(rows){
				var vm = this,
					tabs = 'egwKpiMeasTable',
					rowKey = 'serialNumber';
				vm.selectionData = vm.selectionData.filter((items)=>{
					return items[rowKey] != rows[rowKey]
				});
				var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
					irow= selection.filter((items)=>{
						return items[rowKey] == rows[rowKey]
					})[0];
				vm.$refs[tabs].toggleRowSelection(irow,false);
				var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey])
				vm.$refs[tabs].ckList.splice(idx,1);
			},
			// 打开测量维护任务文件下载页面
			openKpiMeasFilePage(row){
				var vm = this;
				vm.slideUrl = '${ctx}/egw/pm/customizemg/goCustomizeMgmtFilePage.action',
				vm.slideFooter = false;
				vm.slideHeader = false;
				vm.slidePosition = 'top';
				vm.slideHeight = '100%';
				vm.slideWidth = '100%';
				vm.$refs.slideView.showSlide(function(){
					eventBus.$emit("egwkpiMeas-file",row)		
				});
			},
			// 二级页面关闭
			cancelViewSlide(){ 
				var vm = this;
				vm.$refs.slideView.hide();
				vm.$refs['egwKpiMeasTable'].refresh();
			},
	    }
	});
</script>