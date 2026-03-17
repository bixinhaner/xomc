<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo user = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>

<script type="text/javascript" src="${ctx}/js/element/Sortable.min.js?_=${omc_ver}"></script>
<script type="text/javascript" src="${ctx}/js/element/vuedraggable.umd.min.js?_=${omc_ver}"></script>

<style>
	#kpiMeasPage {
		height: 100%;
		display: flex;
		width: 100%;
		position: relative;
		overflow: hidden;
	}
	/*.chengong .el-icon-status-active:before{
		color:#67D972;
		font-size: 20px
	}
	.shibai .el-icon-status-active:before{
		color:#E88282;
		font-size: 20px
	}
	.el-icon-status-conn-off:before ,.el-icon-status-disable:before , .el-icon-status-enable:before{
		font-size: 20px
	}
	.el-icon-circle-warning:before{
		font-size: 20px
	}*/
	#kpiMeasPage .kpiMeasEnableClickBox{
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
	/*.greyIcon::before{
		color: #7A7992;
		font-size: 14px;
	}
	.whirtIcon::before{
		color: #FFFFFF;
		font-size: 14px;
	}*/
	#kpiMeasPage .labelSlotCls > span{
		color: #999999;
		font-size: 12px;
		margin-left: 10px;
	}
	/*.tooltipCls.is-dark{
		background : #959595 ;
		color : #FFFFFF ;
	}
	.tooltipCls[x-placement^=top] .popper__arrow ,
	.tooltipCls[x-placement^=top] .popper__arrow::after{
		border-top-color: #959595!important;
	}

	.tooltipCls[x-placement^=bottom] .popper__arrow ,
	.tooltipCls[x-placement^=bottom] .popper__arrow::after {
		border-bottom-color: #959595!important;
	}
	.tooltipCls[x-placement^=right] .popper__arrow ,
	.tooltipCls[x-placement^=right] .popper__arrow::after {
		border-right-color: #959595!important;
	}
	.tooltipCls[x-placement^=left] .popper__arrow ,
	.tooltipCls[x-placement^=left] .popper__arrow::after {
		border-left-color: #959595!important;
	}*/
	#kpiMeasPage .el-ctable-toolbar{
		padding: 0px !important;
	}
	#kpiMeasPage .el-ctable th>.cell {
		max-height: 25px !important;
	}
	#kpiMeasPage .kpiMeasStatusCls{
		display: flex;
		align-items: center;
	}
	#kpiMeasPage .kpiMeasStatusCls span{
		margin-right: 10px;
	}
	#kpiMeasPage .kpiMeasStatusCls .el-icon::before{
		font-size: 20px;
	}
</style>
<!-- enb测量维护 主页面 -->
<div id="kpiMeasPage" class="commonWarp">
	<el-ctable id="kpiMeasTable" ref="kpiMeasTable" 
		:url="kpiMeasTableUrl"  
		:limit="limitBatch"
		:row-key="'serialNumber'" 
		:query-params="queryKpiMeasParams" 
		@selection-change='batchSelect' 
		:time="6"
		@load-success="tableLoadSuccess" style='width: 100%;'>
		<!-- 高级查询 -- eNb-->
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
										<div><%=rb.getString("YiXuanSheBei")%></div>
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
		<el-table-column v-if="optBtnShow" label='<%=rb.getString("ShiFouQiYong") %>' width="110" prop="report_enable" sortable>
			<template slot-scope="scope">
				<el-switch :ref="scope.row.serialNumber" v-model="scope.row.report_enable" style="height: 18px;margin-left: 10px;"
					active-color="#4D84FF"
					active-value="1"
					inactive-value="0">
				</el-switch>
				<div class="kpiMeasEnableClickBox" @click="kpiMeasEnabelChange(scope.row)"></div>
			</template>
		</el-table-column>
		<el-table-column label='<%=rb.getString("ZhuangTai") %>' width="120" prop="status" sortable>
			<template slot-scope="scope">
				<div class="kpiMeasStatusCls" v-if="scope.row.status === '0'">
					<span class="el-icon el-icon-status-kpi-off "></span><%=rb.getString("Guan") %>
				</div>
				<div class="kpiMeasStatusCls" v-if="scope.row.status === '1'">
					<span class="el-icon el-icon-status-kpi-normal greenIcon"></span><%=rb.getString("ZhengChang") %>
				</div>
				<div class="kpiMeasStatusCls" v-if="scope.row.status === '2'">
					<span class="el-icon el-icon-status-kpi-failure redIcon"></span><%=rb.getString("SuiHuaiZhongZhi") %>
				</div>
			</template>
		</el-table-column>
		<el-table-column label='<%=rb.getString("XiaoZhanBianMa") %>' min-width="120" prop="serialNumber" sortable></el-table-column>
		<el-table-column label='<%=rb.getString("HostName") %>' min-width="120" prop="hostName" sortable></el-table-column>
		<el-table-column label='<%=rb.getString("JiZhanID") %>' min-width="120" prop="cellId" sortable></el-table-column>
		<el-table-column label='<%=rb.getString("CeLiangZhouQi")%>(<%=rb.getString("FenZhongDaXie")%>)' min-width="120" prop="reportPeriod"></el-table-column>
		<el-table-column label='<%=rb.getString("KaiShiShiJian") %>' min-width="120" prop="startTime" sortable></el-table-column>
		<el-table-column label='<%=rb.getString("GengXinShiJian") %>' min-width="120" prop="updateTime" sortable></el-table-column>
	</el-ctable>
			
	 <!-- 二级页面 -- 测量维护文件页面 -->
	 <el-slide ref="slideView" :url="slideUrl" :title="slideTitleView" :footer="false" :header='slideHeader' :position="slidePosition" :force-position="true"
	    :height="slideHeight" :modal='modal' :width="slideWidth" @cancel='cancelViewSlide'>
	</el-slide>
</div>
<script type="text/javascript">
var refreshTable;
if(window.kpiMeasPage) {
	try {
		window.kpiMeasPage.$destroy();
	}catch(e){}
}
window.kpiMeasPage = new Vue({
	el:'#kpiMeasPage',
	data(){
		return{
			kpiMeasTableUrl:'${ctx}/pm/customizemg/getCustomizeListPageData.action',
			queryKpiMeasParams:{
				timeZone: timeZone,
				searchText:'',
				status:'',
				measEnable:'',
			},
			selectionData:[],
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
			
			bulkSelectShow:false,
			searchText:'',
			placeholderText:'<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>',
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
						{value:'1',label:'<%=rb.getString("QiYong")%>'},
						{value:'0',label:'<%=rb.getString("JinYong")%>'}
					],
					value:'measEnable',
				},
			],
			kpiMeasEnableCodes:{
				'0':'0',
				'1':'1',
				'2':'1'
			}
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
	mounted(){
		this.init();
	},
	watch:{},
	methods:{
		init(){

		},
		// 模糊搜索
		query(){
			var vm = this;
			vm.queryKpiMeasParams.searchText = vm.searchText;
			
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
			Object.assign(vm.queryKpiMeasParams, params);
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
			Object.assign(vm.queryKpiMeasParams, params);
			document.body.click();
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
			vm.$refs["kpiMeasTable"].clearSelection();
		},
		// 设备已选表格 单个删除事件
		delBulkSelected(rows){
			var vm = this,
				tabs = 'kpiMeasTable',
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
		
		// 表格加载成功
		tableLoadSuccess(data){
			var vm = this;

		},
		// 批量操作 ———— 测量维护任务 开关
		batchKpiMeasEnable(status){
			var vm = this,
				codeList=[],
				confirmMsg='',
				isReboot = false,
				url='${ctx}/pm/customizemg/setSingleEnbReportKPISwitch.action',
				params = {
					activeReport: status == 'off'? '0' : '1',
					smallCellCodes:'',
				};
			if(vm.selectionData.length <= 0)return
			vm.selectionData.map(function(item,idx){
				codeList.push(item.smallCellCode);
				if(item.needReboot == '1'){
					isReboot = true;
				}
			});
			params.smallCellCodes = codeList.join(',');
			if(status === 'on'){
				if(isReboot){
					confirmMsg = '<%=rb.getString("XuYaoChongQiQueRenKaiQiCeLiangRenWu")%>'
				}else{
					confirmMsg = '<%=rb.getString("QueDingKaiQiCeLiangRenWu")%>'
				}
			}else{
				if(isReboot){
					confirmMsg = '<%=rb.getString("XuYaoChongQiQueRenGuanBiCeLiangRenWu")%>'
				}else{
					confirmMsg = '<%=rb.getString("QueRenGuanBiCeLiangRenWu")%>'
				}
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
						vm.$refs["kpiMeasTable"].clearSelection();
						vm.$refs['kpiMeasTable'].refresh();
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
		cancelViewSlide(){ // 二级页面关闭
	    	var vm = this;
	    	vm.$refs.slideView.hide();
			vm.$refs['kpiMeasTable'].refresh();
	    },
		// 去除首尾空格
		strTrim(str) {
			return str.replace(/(^\s*)|(\s*$)/g, "");
		},
		// 测量维护任务开关改变事件
		kpiMeasEnabelChange(row){
			var vm = this,
				confirmMsg='',
				isReboot = row.needReboot == '1' ? true : false,
				url='${ctx}/pm/customizemg/setSingleEnbReportKPISwitch.action',
				params = {
					activeReport: row.report_enable == '0'? '1' : '0',
					smallCellCodes: row.smallCellCode,
				};
			if(row.report_enable === '0'){
				if(isReboot){
					confirmMsg = '<%=rb.getString("XuYaoChongQiQueRenKaiQiCeLiangRenWu")%>'
				}else{
					confirmMsg = '<%=rb.getString("QueDingKaiQiCeLiangRenWu")%>'
				}
			}else{
				if(isReboot){
					confirmMsg = '<%=rb.getString("XuYaoChongQiQueRenGuanBiCeLiangRenWu")%>'
				}else{
					confirmMsg = '<%=rb.getString("QueRenGuanBiCeLiangRenWu")%>'
				}
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
						vm.$refs["kpiMeasTable"].clearSelection();
						vm.$refs['kpiMeasTable'].refresh();
					}else{
						vm.$message({
							message: data.message,
							type:'error',
						});
					}
		    	}).catch(function(error){})
	    	}).catch()
		},
		// 打开测量维护任务文件下载页面
		openKpiMeasFilePage(row){
			var vm = this;
			vm.slideUrl = '${ctx}/pm/customizemg/goCustomizeMgmtFilePage.action',
			vm.slideFooter = false;
			vm.slideHeader = false;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
			vm.$refs.slideView.showSlide(function(){
				eventBus.$emit("kpiMeas-file",row)		
			});
		},
	},
})
</script>