<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>

<!-- 指标管理 -->
<style type="text/css">
	.new-kpi-indicator-label { width: 150px; display: inline-block; }
	.new-kpi-indicator-label-style { width: 150px; display: inline-block; }
	.new-kpi-inline-div { width: 500px; display: inline-block; }
	.new-kpi-block-div { padding: 0px 0px 25px 0px; }
	.itemDiv { height: 40px; }
	.templateGroupImgBtn { display: inline-block; width: 20px; height: 20px; background-position: center center; float: right; margin-right: 5px; }
	.iconText li a { color: #1DA3FC; }
	#kpiAlgorithmicOperDiv .windowButtonGroup .linkbutton>span { min-width: 55px; }
	
	#kpiManagePage .tree-file { background: unset; margin: 0; width: 0;}
	#kpiManagePage .tree-join, #kpiManagePage .tree-indent { width: 0;}
	.tree-root-one  { display: none; }
	.selectedDetail_mana { height: 24px; line-height: 24px; border: 1px solid #CCE1EF; background: #E1F2FA; margin-right: 10px; margin-bottom: 8px; display: inline-block; color: #949494; }
	.flexLayout { display: flex !important; height: 100%; }
	.flexLeft { flex-shrink: 0; flex-grow: 0; -webkit-flex-shrink: 0; -webkit-flex-grow: 0; }
	.flexRight { flex-shrink: 1; flex-grow: 1; -webkit-flex-shrink: 1; -webkit-flex-grow: 1; }
	.omcTableTool { padding: 20px 3px; }
	.forbidden > span { background: #B0CBDD !important; }
	.kpi-text-node { cursor: default; display: inline-block; padding: 0px 9px; color: #979FA6; border: 1px solid #DCDFE6; background: #EFF0F3; height: 18px; line-height: 18px; margin-bottom: 5px; }
	#kpiAlgorithmicOperNameDiv:empty { display: none; }	
	.groupMgmt { flex: 300px 0 0;border-right: 1px solid #E9E9E9; }
	.deviceList { flex: 1; }
	.splitTitle { font-size: 14px; font-weight: bold; color: #363B4E; padding: 15px 0 15px 10px; }
	.kpiSymbol { padding: 6px; width: 768px; border: 1px solid #BADCFE; background: #F1F7FE; }
	.kpiSymbol .linkbutton { min-width: 56px; }
	.addKpiBtn { font-size: 26px;}
	#kpiManagePage .el-dropdown-menu__item { text-align: left; font-weight: 600;}
	#kpiManagePage .el-dropdown-menu { left: 50px; width: 160px;}
	#kpiManagePage .kpiTree .tree-title { width:calc(100% - 26px); padding-left: 20px;}
	
	.slideTitle { font-size: 14px; font-weight: bold; color: rgba(0, 0, 0, 0.8); padding: 0 20px; height: 46px; line-height: 46px; border: none; }
	.datagrid-mask {
		opacity: 0;
		filter: alpha(opacity=0);
	}
	.datagrid-mask-msg {
		opacity: 0;
		filter: alpha(opacity=0);
	}
		
	#kpiManagePage .enbEgwKpiTree .tree>li>div>span.tree-title {
		width: calc(100% - 46px);
		padding-left: 0px;
	}
	#kpiManagePage .enbEgwKpiTree .tree>li>div {
		padding-left: 10px;
	}
	#kpiManagePage .enbEgwKpiTree {
		overflow: auto;
	}
	#kpiManagePage .enbEgwKpiTree li ul { 
		
	}
	#kpiManagePage .enbEgwKpiTree li ul li div .tree-title {
 		padding-left: 20px; 
		width: calc(100% - 26px);
	}
</style>
<div class="overflow-cls">
<!-- KPI管理界面 -->
<div class="commonFlex" id="kpiManagePage" style="height: 100%; min-width: 1200px;position: relative;width: 100%;overflow: hidden;">
	<div class='commonFlex' style='width: 100%; height: 100%;'>
		<div class='commonWarp' style='width: 100%; height: 100%; display: flex; flex:1; position: relative; background: #fff;'>
			<!-- 右上角添加按钮 -->
			<div class="newIconBoxCls-bt CODE_PERFORMANCE_MANAGEMENT hidden" style="top:6px;right:50px;">
				<span class="el-icon el-icon-plus" onclick="goAddKpiArithmeticWin()"></span>
				<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
			</div>
			<div class="newIconBoxCls-bt" style="top:6px; right: 10px">
				<span class="el-icon el-icon-operation-export" onclick="exportKpiInfo()"></span>
				<div class="titleButtonText"><%=rb.getString("DaoChu")%></div>
			</div>
			{{currentKpiNetType}}
			<div v-if="currentKpiNetType == 'enb'" style='width: 240px; flex: 240px 0 0; border-right: 1px solid #D5DCEC;'>
				<div class="slideTitle" style='font-size: 14px; border-bottom: 1px solid #D5DCEC;font-weight: bold; color: rgba(0, 0, 0, 0.8); padding: 0 20px; height: 46px; line-height: 46px;'><%=rb.getString("ZhiBiaoGongNengJi")%></div>
		
				<div class="enbEgwKpiTree">
					<div id="kpiArithmeticTree" style='height: calc(100% - 46px);'></div>
				</div>
			</div>
			<div v-else style='width: 240px; flex: 240px 0 0; border-right: 1px solid #D5DCEC;'>
				<div style="position: relative;">
					<div class="newIconBoxCls-bt CODE_PERFORMANCE_MANAGEMENT hidden" style="top:12px;right:10px;">
						<span class="el-icon el-icon-plus addKpiBtn" @click="goAddKpiGroup('')"></span>
					</div>
					<div class="slideTitle" style='font-size: 14px; border-bottom: 1px solid #D5DCEC;font-weight: bold; color: rgba(0, 0, 0, 0.8); padding: 0 20px; height: 46px; line-height: 46px;'><%=rb.getString("ZhiBiaoGongNengJi")%></div>
				</div>
				<div class="kpiTree">
					<div id="kpiArithmeticTree" style='height: calc(100% - 46px); overflow: auto;'></div>
				</div>
			</div>
			<div style='flex: 1; '>
				<table id="kpiDatagrid" style='width: 100%; '></table>
			</div>
		</div>
			
		<div class='rightWarp' style='position: relative; flex: 0 1 360px;' v-show='addFunNameShow '>
			<div class='rightWarpLayer'>
		 		<div class='rightBoxHeaderHasTip'>
					<div class='headerText'>
						<span>{{enbOrGnbTitle}}</span>
						<span class='closeIconBox' @click='closeKpiGroupFun'><i class='el-icon el-icon-close'></i></span>
					</div>
				</div>
		 		<div class='addApnSlide rightWarpLayerContent' style='padding: 20px;'>
		 			<el-form ref="addFunNameForm" label-position="top" :model="addFunNameForm" :rules="addFunNameRules">
		 				<el-form-item :label='commonName' prop='catagoryName'>
							<el-input v-model='addFunNameForm.catagoryName' style='width: 300px'></el-input>
						</el-form-item>
						<el-form-item label='<%=rb.getString("XiangXiMiaoShu")%>' prop='description'>
							<el-input type='textarea' :rows="3" v-model='addFunNameForm.description' style='width: 300px'></el-input>
						</el-form-item>
		 			</el-form>
		 		</div>
				<div class='commonFlex commonBorderTop commonFormFotter'>
					<div>
						<el-button type="primary" @click="addKpiGroupFun"><%=rb.getString("QueDing")%></el-button>
						<el-button @click="closeKpiGroupFun"><%=rb.getString("QuXiao")%></el-button>
					</div>
				</div>
			</div>
		</div>
	</div>	
		
	<!-- 指标功能集新建及修改 -->
	<div id="kpiManaTemplate" class="slidebarPanel" style="width:1000px;"></div>
	<!-- 指标新建及修改 -->
	<div id="kpiManaAdd" class="slidebarPanelTop slide-position-top slidebarPanel" style="position:absolute;"></div>
	<div id="kpiManaAddOrModify" class="slidebarPanel" style="width:1000px;position:absolute;"></div>
	<!-- 工具栏-设备组下基站列表 -->
	<div id="kpiDatagridToolbar" class="toolbarContainer" style="display:none; padding: 0 !important;">
		<div class='toolbarHeadBtnBoxCls' style='border-top: 0; margin-bottom: 0; height: 36px;'>
			<span class='commonText14' style='margin: 0 10px;'>{{selectKpiName}}</span>			
            <!-- 已选数据 只有enb 显示-->
            <div v-if="currentKpiNetType == 'enb' && writableMap['CODE_PERFORMANCE_MANAGEMENT'] == true" class='commonFlex'>
            <div class="selectBlukBoxCls">
                <div class="selectMain" style="padding-top: 7px;">
                    <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
                        <span class="el-icon-selected el-icon"></span>
                        <span class="bulkSelectNumBoxCls">( {{selectedRows.length}} )</span>
                    </div>
                    <div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="bulkSelectShow">
                        <div class="selectBoxTitle">
                            <span><%=rb.getString("YiXuan")%></span>
                            <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
                        </div>
                        <div class="selectBoxMain">
                            <div class="tableInfoCls">
                                <div class="tableInfoHeader">
                                    <div><%=rb.getString("ZhiBiaoID")%></div>
                                    <div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span><%=rb.getString("QingChu")%></div>
                                </div>
                                <el-ctable 
                                    id="bulkSelectTable" 
                                    ref="bulkSelectTable" 
                                    :data="selectedRows" 
                                    :showHeader="false"
                                    :rownumber="false"
                                    :front-pagination="true"
                                    height="270px" pagination="true" >
                                    <el-table-column prop="id" v-if="false"></el-table-column>
                                    <el-table-column width="588">
                                        <template slot-scope="scope" >
                                            <div class="tableItemCls">
                                                <span>{{scope.row.kpiId}}</span>
                                                <span @click="delBulkSelected(scope.row, scope.$index)" class="el-icon el-icon-circle-close item_show"></span>
                                            </div>
                                        </template>
                                    </el-table-column>
                                </el-ctable>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            <div :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" onclick='enableBulkClick()'>
                <span class='el-icon el-icon-KPI-Meas'></span>
                <span><%=rb.getString("CeLiangNew")%></span>
            </div>
            <div :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" onclick='disableBulkClick()'>
                <span class='el-icon el-icon-operation-CancelMeasure'></span>
                <span><%=rb.getString("QuXiaoCeLiang")%></span>
            </div>
            </div>
        </div>
		<div class='commonFlex' style='padding: 10px;'>
			<div class="queryGroup" style='height: 24px; margin: 0 10px;'>
				<input id="kpimgt_serach_text" placeholder="<%=rb.getString("ZhiBiaoMingCheng")%> / <%=rb.getString("ZhiBiaoID")%>" style="width: 240px; height: 24px;" /> 
				<b class="el-icon el-icon-common-search" onclick="kpiArithmeticTreeLoad()"></b>
			</div>
			<div class='commonFlex' v-if="currentKpiNetType == 'enb'">
				<el-popfilter
					type="single"
					label='<%=rb.getString("ChanPinLeiXing")%>'
					v-model="productSelectParam"
					:list="productOptions.map(function(item){ return {label: item.name, value: item.value}})"
					@check-change="productChange">
				</el-popfilter>
				<el-popfilter style="margin-left: 10px;"
					type="single"
					label='<%=rb.getString("DengJi")%>'
					v-model="levelSelectParam"
					:list="levelOptions.map(function(item){ return {label: item.text, value: item.value}})"
					@check-change="levelChange">
				</el-popfilter>
	     	</div>
		</div>
	</div>
	<!--指标对应结果提示 -->
	<el-dialog title="<%=rb.getString("QueRen")%>" :visible.sync="kpiResultTipDialog" width="660" :close-on-click-modal="false" top="30vh" @close="kpiResultTipDialog = false">
        <div v-if='operOrMainPageFlag == true' class='commonSize14' style='margin-bottom: 20px;'><%=rb.getString("QueDingJinYong")%></div>
        <el-ctable v-if='kpiTableData.length > 0' id='kpiResultTable' ref="kpiResultTable" :data='kpiTableData' height='320px' :pagination="false" :rownumber=false style='border: 1px solid #D5DCEC;'>
            <el-table-column prop="kpiId" label="<%=rb.getString("ZhiBiaoID")%>"></el-table-column>
            <el-table-column prop="result" label="<%=rb.getString("JieGuo")%>">
                <template slot-scope="scope">
                    <div><%=rb.getString("ZhiBiaoYiBeiMuBanGuanLian")%></div>
                </template>
            </el-table-column>
        </el-ctable>
        <span slot="footer" class="dialog-footer" v-if='operOrMainPageFlag == true'>
            <div class="buttonGroup autoUpdateDialog">
                <el-button type="primary" @click="kpiMeasSubmit"><%=rb.getString("QueDing")%></el-button>
                <el-button @click="kpiResultTipDialog = false"><%=rb.getString("QuXiao")%></el-button>
            </div>
        </span>
    </el-dialog>

	<el-dialog top="30vh" width="360"
		:visible.sync="duplicateShow" 
		:close-on-click-modal="false"
		@close="closeDuplicateDl"
	>
		<div>
			<%=rb.getString("ChongMingTiShiPre")%> {{duplicateForm.kpiId}}, 
			<div><%=rb.getString("ChongMingTiShiAfter")%></div>
			<el-input v-model="duplicateForm.indicatorName" maxlength="50"></el-input>
		</div>
        <span slot="footer" class="dialog-footer">
            <div class="buttonGroup autoUpdateDialog" style="text-align: right;">
                <el-button type="primary" @click="saveDuplicate"><%=rb.getString("QueDing")%></el-button>
                <el-button @click="closeDuplicateDl"><%=rb.getString("QuXiao")%></el-button>
            </div>
        </span>
	</el-dialog>
</div>
</div>
<!-- 导出所有指标用的表单 -->
<form method="post" style="display: none"  id="exportAllKpiForm"></form>

<script>   
var rootId = "", curEnbOrGnbTreeUrl1 = '';
if(window.kpiManagePageVue) {
	try {
		window.kpiManagePageVue.$destroy();
	}catch(e){}
}
window.kpiManagePageVue = new Vue({
		el:"#kpiManagePage",
		data() {
			var validFunName = function(rule,value,callback){			  
				if(value === null || value === "" || value === undefined){
					callback(new Error("<%=rb.getString("QingShuRuZhiBiaoGongNengJiMingCheng")%>"));
				}else{
					callback();
				}
			}
		    return {
				lastNeType: '',  // 记录上一次的网元类型，用于判断网元是否真正改变
				duplicateShow: false,
				duplicateForm: {
					kpiId: '',
					indicatorName: ''
				},

		    	enbOrGnbTitle:'',
				commonName:'',
		    	addFunNameShow: false,
				enbOrGsmTreeAddType: '',
		    	addFunNameForm:{				
		    		catagoryName: '',
		    		description: ''
				}, 
				addFunNameRules: {
					catagoryName: [{validator: validFunName}]						
				},
				curFunNameType: 'add',
				curModifyFunNameId: '',
				selectedRows: [],
				selectKpiName: '',
				bulkSelectShow: false,
				productSelectParam: '',
				productOptions: [],
				levelSelectParam: '',
				levelOptions: [
				    { text: '<%=rb.getString("QuanBu")%>', value: '' },
					{ text: 'Device', value: 'device' },
				    { text: 'PLMN', value: 'plmn' },
				],
				kpiResultTipDialog: false,
				kpiTableData:[],
				//区分属于 点击批量操作的取消测量还是修改页面的取消测量
				operOrMainPageFlag: false
		    }
		},
		computed:{
			// 合并后的唯一网元标识计算属性
			currentKpiNetType() {
				let netTypeValue = '';
				//遍历 sysMain.$refs.nav.editableTabs 获取当前所有打开的tab
				if(sysMain.$refs.nav && sysMain.$refs.nav.editableTabs && sysMain.$refs.nav.editableTabs.length > 0){
					sysMain.$refs.nav.editableTabs.map(function(item,index){
						//根据 id 判断是否是 4G、5G、eGW 网元
						if(item.id == '900003' || item.id == '900006' || item.id == '900008'){
							netTypeValue = item.netType;
						}					
					});
				}
				
				return netTypeValue && netTypeValue != '' ? netTypeValue : sysMain.headType;
			},
		},
		watch:{
			currentKpiNetType(newType, oldType) {
				var vm = this;
				
				// 当前功能仅支持 eNB、gNB、eGW 网元的指标管理
				if(newType !== 'enb' && newType !== 'gnb' && newType !== 'egw') {
					// 不支持的网元：什么都不做，保持页面当前状态
					return;
				}

				// 如果网元类型没有真正改变（如 eNB → CPE → eNB），不需要重新初始化
				if(vm.lastNeType === newType) {
					
					return;
				}

				// 支持的网元且类型确实改变了：重新初始化
				vm.lastNeType = newType;
				
				// 只有 4g 网元时，显示产品类型下拉框
				if(vm.currentKpiNetType == 'enb'){
					vm.kpiInit();
					//切换网元，清空已选指标
					vm.selectedRows = [];
					$('#kpiDatagrid').datagrid('clearChecked');
				}
				
				closeLoading();
				
				//初始化或切换 4G,5G时需刷新左侧指标功能集及右侧指标列表
				$("#kpimgt_serach_text").val(''); //清空搜索框
				kpiManasearchText = '';
								
			    if(vm.currentKpiNetType == 'enb'){
				    curEnbOrGnbTreeUrl1 = '${ctx}/pm/indicatormg/getIndicatorGroupTree.action';
		        }else if(vm.currentKpiNetType == 'gnb'){
					curEnbOrGnbTreeUrl1 = '${ctx}/gnb/pm/indicatormg/getIndicatorGroupTree.action';
				}else if(vm.currentKpiNetType == 'egw'){
					curEnbOrGnbTreeUrl1 = '${ctx}/egw/pm/indicatormg/getIndicatorGroupTree.action';
				}

				vm.eNBOrGnbTreeChange();

				vm.addFunNameShow = false;
				vm.$nextTick(() => {
					try{
						// 只 resize 当前页面的 datagrid，避免影响其他页面（如 KPI View）
						vm.kpiResizeDatagrid();
						//$('#kpiManagePage table.datagrid-f').datagrid('resize');
						window.dispatchEvent(new Event('resize'));
					}	catch(e){}
				})
			}
		},
		methods:{
			// 辅助方法：只 resize 当前页面的 datagrid，避免影响其他页面
			kpiResizeDatagrid() {
				try {
					$('#kpiManagePage table.datagrid-f').datagrid('resize');
				} catch(e) {
				}
			},
		    //确定取消测量
		    kpiMeasSubmit(){
                var vm = this, curCheckList = $('#kpiDatagrid').datagrid('getChecked');

                    vm.kpiResultTipDialog = false;
                    vm.operOrMainPageFlag = false;
                if(curCheckList.length > 0){
                    var curIndicatorIds = curCheckList.map(function(item){ return item.kpiId}).join(',');
                    var params = {
                        indicatorIds: curIndicatorIds
                    };
                }else{
                    return;
                }
                $.post("${ctx}/cell/perfmgmt/kpimanage/disableIndicator.action", params, function (data) {
                    if(data["success"]){
                        cancleEnbaleClick();
                    } else {
                        showMsg('error_msg',data["msg"]);
                    }
                }, "json");
            },
			eNBOrGnbTreeChange(){
				var vm = this;
	    	   	setTimeout(function(){
		    	   $("#kpiArithmeticTree").tree({
			           url: curEnbOrGnbTreeUrl1,
			           animate: true,
			           lines: true, // 是否显示树形结构中的树线
			           border: true,
			           onBeforeLoad :function(node,param){
			        	   if(kpiManasearchText.length == ""){
				        	   param["noKpiShowGroup"]=1;
			        	   }else{
				        	   param["noKpiShowGroup"]=0;
			        	   }
			        	   param["searchText"]= kpiManasearchText;
			        	   
			        	   if(vm.currentKpiNetType == 'enb'){
			        		   param["product_type"] = tableProductTypeSelect;
								//level 功能集查询不需要这个参数
							  // param["indicatorLevel"] = vm.levelSelectParam;
			        	   }else if(vm.currentKpiNetType == 'gnb'){
			        		   //5g 参数 20220812: 暂时将 5g,egw 网元下涉及的产品类型参数隐藏；
			        		   //param["product_type"]= 'BBU_XSS';
							   //param["indicatorLevel"] = vm.levelSelectParam;
			        	   }else{
			        		   //eGW 参数  53047 注意这里的产品类型
			        		   //param["product_type"]= 'WCG';
			        	   }
			        	  
			        	   param["timeZone"]= timeZone;
			           },
			           onLoadSuccess:function(node,data){
			        	   
			               $("#kpiArithmeticTree li:eq(1)").find("div").addClass("tree-node-selected");
			               var n = $("#kpiArithmeticTree").tree("getSelected");  
			              
			               if(n!=null){ 
			                    $("#kpiArithmeticTree").tree("select",n.target);    
			               }else{
			            	   $('#kpiDatagrid').datagrid('loadData',[]);
			            	   //筛选的传参
			                   globalQueryParams = {
			                   		catagoryId: "",
			    		   			searchText: kpiManasearchText,
			    		   			timeZone: timeZone,
			                   };
			                   if(vm.currentKpiNetType == 'enb'){
			                	   globalQueryParams.product_type = tableProductTypeSelect;
								   //level
								   globalQueryParams.indicatorLevel = kpiManagePageVue.levelSelectParam;
				        	   }else if(vm.currentKpiNetType == 'gnb'){
				        		   //5g 参数
				        		   //globalQueryParams.product_type= 'BBU_XSS';
				        	   }else{
				        		   //eGW 参数
				        		   //globalQueryParams.product_type= 'WCG';
				        	   }
			               }
			            },
			           onBeforeSelect:function(node){
			        	   if(!$(this).tree("isLeaf",node.target)){
			        		   return false;
			        	   } 
			           },
			           onSelect: function (node){
							var curUrl = '',
				        	    params = {
									catagoryId: node.id,
									searchText: kpiManasearchText,
									timeZone: timeZone,
								};
							
			        	    //筛选的传参
			                globalQueryParams = {
			               		catagoryId: node.id,
					   			searchText: kpiManasearchText,
					   			timeZone: timeZone,
			                };
			                kpiManagePageVue.selectKpiName = node.text;
			        	    
							if(vm.currentKpiNetType == 'enb'){
								//置空已选的指标数据
								//kpiManagePageVue.selectedRows = [];
								//$('#kpiDatagrid').datagrid('clearChecked');

								curUrl = '${ctx}/pm/indicatormg/getIndicatorListByPage.action';
		            		    params.product_type = tableProductTypeSelect;
		            		    globalQueryParams.product_type = tableProductTypeSelect;
								//level
								params.indicatorLevel = kpiManagePageVue.levelSelectParam;
								globalQueryParams.indicatorLevel = kpiManagePageVue.levelSelectParam;
		            	    }else if(vm.currentKpiNetType == 'gnb'){
		            		    curUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorListPageData.action';
		            		    //params.product_type = 'BBU_XSS';
		            		    //globalQueryParams.product_type = 'BBU_XSS';
		            	    }else{
		            	    	 curUrl = '${ctx}/egw/pm/indicatormg/getIndicatorListPageData.action';
			            		 //params.product_type = 'WCG';
			            		 //globalQueryParams.product_type = 'WCG';
		            	    }

			                if(node.id != rootId){
			                	$('#kpiDatagrid').datagrid({
				                    queryParams: params,
				                    url: curUrl
				                });
			                };		              
			           },
			           onContextMenu:function(e){
			              e.preventDefault();
			           },
			           formatter : operateKpiTree
			      }); 
	    	   },200);
	       },
			
			
			addKpiGroupFun(){
				var vm = this, curSaveUrl = '';
				
				if(vm.curFunNameType == 'add'){
					var params = {
						"catagoryName": vm.addFunNameForm.catagoryName,
						"description" : vm.addFunNameForm.description
					};
					if(vm.currentKpiNetType == 'enb'){
						params.device_type = vm.enbOrGsmTreeAddType; //添加需要
						curSaveUrl = '${ctx}/pm/indicatormg/addIndicatorGroup.action';
					}else if(vm.currentKpiNetType == 'gnb'){
						curSaveUrl = '${ctx}/gnb/pm/indicatormg/addIndicatorGroup.action';
					}else{
						curSaveUrl = '${ctx}/egw/pm/indicatormg/addIndicatorGroup.action';
					}
				}else{
					//node: node=$('#kpiArithmeticTree').tree("getSelected") 只可获取到 name 和 id 的值；
					// 从接口可以获取到 描述的值；
					var params = {
						"catagoryId": vm.curModifyFunNameId,
						"catagoryName": vm.addFunNameForm.catagoryName,
						"description" : vm.addFunNameForm.description || ''
					};
					if(vm.currentKpiNetType == 'enb'){
						curSaveUrl = '${ctx}/pm/indicatormg/modifyIndicatorGroup.action';
					}else if(vm.currentKpiNetType == 'gnb'){
						curSaveUrl = '${ctx}/gnb/pm/indicatormg/modifyIndicatorGroup.action';
					}else{
						curSaveUrl = '${ctx}/egw/pm/indicatormg/modifyIndicatorGroup.action';
					}
				}
				console.log(vm.enbOrGsmTreeAddType)
				vm.$refs.addFunNameForm.validate((valid) => {
					if(valid ){
						axios.post(curSaveUrl,stringify(params)).then(function(response){
		    				var data = response.data;
		    				if(data["success"]){
		    					vm.$message({
									message:"<%=rb.getString("ChengGong")%>",
									type:'success',
								});
		    					$("#kpiArithmeticTree").tree("reload");
								vm.addFunNameShow = false;
		    				}else{
		    					vm.$message.error(data["message"])
		    				}
		    			});
					}
				});
				
				vm.$nextTick(() => {
					try{
						vm.kpiResizeDatagrid();
						//$('table.datagrid-f').datagrid('resize');
						window.dispatchEvent(new Event('resize'));
					}	catch(e){}
				})
			},
			 
			closeKpiGroupFun(){
				 var vm = this;
				 vm.$refs.addFunNameForm.resetFields();
				 vm.addFunNameShow = false;
				 
				 vm.$nextTick(() => {
					try{
						vm.kpiResizeDatagrid();
						//$('table.datagrid-f').datagrid('resize');
						window.dispatchEvent(new Event('resize'));
					}	catch(e){}
				})
			},
			kpiInit(){
				var vm = this;
				
				axios.post('${ctx}/cell/cpeinfos/getEnbMonitorProductList.action?isGnb=0&isAll=1').then(function(response){
					var data = response.data;
					
					// 动态删除 BTS
					data = data.filter(function(item) {
						return item !== 'BTS';
					});
					newData = data;
					if(data.length == 0){
						tableProductTypeSelect = 'no';
					}else{
						//默认全选时拼接 ALL
						tableCurProduct = 'ALL,' + data.join(',');
						tableProductTypeSelect = tableCurProduct;
					}
					var arr = [{name:'<%=rb.getString("QuanBu")%>',value:''}];
					data.map(function(item){
						if (item){
							arr.push({name:item,value:item})
						}
					})
					vm.productOptions = arr;

				}).catch(function(error){})
                
			 },
			//添加功能集
			goAddKpiGroup(type){
				var vm = this;

				if(vm.currentKpiNetType == 'enb'){
					vm.enbOrGsmTreeAddType = type;

					if(type == 'ENB'){
						vm.enbOrGnbTitle = '<%=rb.getString("XinJianENBZhiBiaoGongNengJi")%>';
						vm.commonName = '<%=rb.getString("ENBZhiBiaoGongNengJiMingCheng")%>';
					}else if (type == 'GSM'){
						vm.enbOrGnbTitle = '<%=rb.getString("XinJianGSMZhiBiaoGongNengJi")%>';
						vm.commonName = '<%=rb.getString("GSMZhiBiaoGongNengJiMingCheng")%>';
					}
				}else{
					vm.enbOrGnbTitle = '<%=rb.getString("XinJianZhiBiaoGongNengJi")%>';
					vm.commonName = '<%=rb.getString("ZhiBiaoGongNengJiMingCheng")%>';
				}
				
				vm.curFunNameType = 'add';
				vm.addFunNameForm.catagoryName = '';
				vm.addFunNameForm.description = '';
				vm.addFunNameShow = true;
				//表格自适应
				vm.$nextTick(() => {
					try{
						vm.kpiResizeDatagrid();
						//$('table.datagrid-f').datagrid('resize');
						window.dispatchEvent(new Event('resize'));
					}	catch(e){}
				})
			},
			// 打开已选弹窗
             openBulkSelectTable(){
                 var vm = this;
                 vm.bulkSelectShow = true
             },
             // 关闭已选弹窗
             closeBulkSelectTable(){
                 var vm = this;
                 vm.bulkSelectShow = false;
             },
             // 设备已选表格 清空事件
             clearBulkSelected(){
                var vm = this;
                vm.selectedRows = [];
				$('#kpiDatagrid').datagrid('clearChecked');
             },
             // 设备已选表格 单个删除事件
             delBulkSelected(row, index){
                 var vm = this;
                 
                 //删除已选数据中的数据
                 kpiManagePageVue.selectedRows.splice(index,1);
                 //清除表格被选中的状态
                 var index = $("#kpiDatagrid").datagrid('getRowIndex',row.kpiId);
                 $('#kpiDatagrid').datagrid('uncheckRow',index);
             },
            	//产品类型改变更新列表
			productChange(val){
            	tableProductTypeSelect = val;
   	    		if(tableProductTypeSelect == ''){
   	    			tableProductTypeSelect = tableCurProduct;
   	    		}
   	    		$("#kpiArithmeticTree").tree("reload");
 			},
			levelChange(val){
				var vm = this;

				vm.levelSelectParam = val;
				//new 只更新指标数据，不影响指标功能集
				//$('#kpiDatagrid').datagrid('reload');
				//与产品类型保持一致
				$("#kpiArithmeticTree").tree("reload");
				try{
					vm.kpiResizeDatagrid();
					//$('table.datagrid-f').datagrid('resize');
					window.dispatchEvent(new Event('resize'));
				}	catch(e){}
			},
			openDuplicateDl() {
				var vm = this;

				vm.duplicateShow = true;
			},
			closeDuplicateDl() {
				var vm = this;

				vm.duplicateShow = false;
				Object.assign(vm.duplicateForm, {
					kpiId: '',
					indicatorName: ''
				})
			},
			saveDuplicate() {
				var vm = this,
					saveUrl = '', 
					params = vm.duplicateForm;
				
				if(vm.currentKpiNetType == 'enb'){
					saveUrl = '${ctx}/pm/indicatormg/updateEnbIndicatorsName.action';
				}else if(vm.currentKpiNetType == 'gnb'){
					saveUrl = '${ctx}/gnb/pm/indicatormg/updateGnbIndicatorsName.action';
				}else{
					saveUrl = '${ctx}/egw/pm/indicatormg/updateBaseKpiCustName.action';
				}
				
				$.post(saveUrl, params, function (data) {
					if (data["success"]) {
						showMsg('success_msg','<%=rb.getString("ChengGong")%>');
						$('#kpiDatagrid').datagrid('reload');
						vm.closeDuplicateDl();
					} else {
						showMsg('error_msg',data["message"]);
					}
				}, "json");
			}
		 },
		mounted(){
			// 记录初始网元类型
			this.lastNeType = this.currentKpiNetType;
			
			//enb才有产品类型
			if(this.currentKpiNetType === 'enb') {
				this.kpiInit();
			}
		}
	})
</script>
<script type="text/javascript">
var QingXuanZeSheBei = '<%=rb.getString("QingXuanZeSheBei")%>';
var QingXuanZeZhiBiao = '<%=rb.getString("QingXuanZeZhiBiao")%>';

var XiuGai='<%=rb.getString("XiuGai")%>';
var ShanChu = '<%=rb.getString("ShanChu")%>';
var TianJia = '<%=rb.getString("TianJia")%>';

var QingShuRuHeFaShuZi = '<%=rb.getString("QingShuRuHeFaShuZi")%>';
var ZhiBiaoMenXianShuZhiSheZhiBuHeLi = '<%=rb.getString("ZhiBiaoMenXianShuZhiSheZhiBuHeLi")%>';
var ZhiBiaoMenXianHuChi='<%=rb.getString("ZhiBiaoMenXianSheZhiHuChi")%>';
var id='';
var kpiManasearchText = '';
var searchTextKPIZhiBiao = '';//新建页面搜索
var addOrModifyKpiLevel = ''; // 新建页面的等级标识
var searchProductTypeSelect = '';//新建页面产品类型搜索
var curProduct = '';
var modifyOrViewKpiFlag = false; //区分基础指标是修改还是查看
var tableProductTypeSelect = '', newData = [];
var curEnbOrGnbTreeUrl = '', curOnClickRowId = '', tableCurProduct = ''; 
var intersetionTip = false;
//var updateTableDataTimer;
var strRandom = Math.random().toString();

var singleArr = [false];
/* 鼠标点击其他位置下拉框隐藏  */
$("#kpiManagePage").click(function(){
    $('.color').hide();
}) 

/* rgb格式转换为十六进制颜色值  */
$.fn.getHexBackgroundColor = function() {
    var rgb = $(this).css('background-color');
    var  ie = /msie/.test(navigator.userAgent.toLowerCase());
    if(!ie){
        rgb = rgb.match(/^rgb\((\d+),\s*(\d+),\s*(\d+)\)$/);
        function hex(x) {
            return ("0" + parseInt(x).toString(16)).slice(-2);
        }
        rgb= "#" + hex(rgb[1]) + hex(rgb[2]) + hex(rgb[3]);
    }
    return rgb;
}

var globalQueryParams={};
/* 筛选展示逻辑处理 */ 
var node=$('#kpiArithmeticTree').tree("getSelected");

	var rules = [
		  {
			title: '<%=rb.getString("Type")%>',
		    match: function(code){
		      return code == 'isCustomize';
		    },
		    action: function(code,e){
		    	var params = {}, data = [], key = 'isCustomize';
		    	params = $('#kpiDatagrid').datagrid('options').queryParams;
		      	/* 配置筛选菜单可选项，可以通过接口获取数据 */
		      	
		        if(kpiManagePageVue.currentKpiNetType == 'enb'){
		        	var url = '${ctx}/pm/indicatormg/getIndicatorTypes.action';
		        }else if(kpiManagePageVue.currentKpiNetType == 'gnb'){
		        	var url = '${ctx}/gnb/pm/indicatormg/getIndicatorTypes.action';
		        }else{
		        	var url = '${ctx}/egw/pm/indicatormg/getIndicatorTypes.action';
		        }
		        
		      	$.post(url,globalQueryParams,function(json){
		      		if(json && json[0] && json[0].isCustomize){
		      			json[0].isCustomize.map(function(item){
		      				var row = {name:'isCustomize',label:item.text,value:item.value};
		      				data.push(row);
		      			});
		      		}
		          	data.map(function(item){
		          		if(params[key]){
		          			var vals = params[key].split(',');
		          			if(vals.includes(item.value+'')) item.checked = true;
		          		}else{
		          			item.checked = true;
		          		}
		          	});
		          	/* 生成筛选菜单 */
		          	filterMenu({
		            	data: data,
		            	fn: function(tips){
		              		tips.css({left:e.x-$('#menuAnimate').width(),top:83});
		              		$('#omc_app_ctn').append(tips);
		            	},
		            	click: function(values){
		            		if(values.indexOf(',')>0){
			            		params[key] = "";
		            		}else{
			            		params[key] = values;
		            		}
		            		$('#kpiDatagrid').datagrid('reload');
		            	}
		          	});
		      	},'json');
		    }
		}
	]; 

	
    $.extend($.fn.validatebox.defaults.rules,{
       checkNumber:{
           validator:function(value,param){
        	   var threadFormData={};
        	    var threadArray=$("#kpiThresholdForm").serializeArray();
        	    var kpiIdUnitValue =  $("#kpiIdUnitValueId").combobox("getValue");
        		 $.each(threadArray,function(index,obj){
        			 if(obj.value==null||obj.value==""){
        				threadFormData[obj.name]=null;
        			 }else{
        				threadFormData[obj.name]=obj.value;
        			 }
        		 })
        		 if(JSON.stringify(threadFormData)!="{}"){
        			 if(kpiIdUnitValue=="%" && (parseFloat(threadFormData.generalBegin) > 100 || parseFloat(threadFormData.generalEnd) > 100
        					|| parseFloat(threadFormData.seriousBegin) > 100 || parseFloat(threadFormData.seriousEnd) > 100)){
        				 $.fn.validatebox.defaults.rules.checkNumber.message=ZhiBiaoMenXianShuZhiSheZhiBuHeLi
        				 return;
        			 }
        			 var boolobj = {},
        			 	 b1 = (parseFloat(threadFormData.seriousBegin) >= parseFloat(threadFormData.generalBegin) && parseFloat(threadFormData.seriousBegin) < parseFloat(threadFormData.generalEnd)),
        			 	 b2 = (parseFloat(threadFormData.seriousEnd) > parseFloat(threadFormData.generalBegin) && parseFloat(threadFormData.seriousEnd) <= parseFloat(threadFormData.generalEnd)),
        			 	 b3 = (parseFloat(threadFormData.generalBegin) >= parseFloat(threadFormData.seriousBegin) && parseFloat(threadFormData.generalBegin) < parseFloat(threadFormData.seriousEnd)),
        			 	 b4 = (parseFloat(threadFormData.generalEnd) > parseFloat(threadFormData.seriousBegin) && parseFloat(threadFormData.generalEnd) <= parseFloat(threadFormData.seriousEnd));
        			 	 b5 = (parseFloat(threadFormData.generalBegin) < parseFloat(threadFormData.generalEnd) && parseFloat(threadFormData.seriousBegin) <= parseFloat(threadFormData.seriousEnd) );
        			 boolobj['seriousBegin']=b1;
        			 boolobj['seriousEnd']=b2;
        			 boolobj['generalBegin']=b3;
        			 boolobj['generalEnd']=b4;
        			 
        				 if(b1 || b2 || b3 || b4 ){
              		 	   var bool = (b1||b2)&&(b3||b4);
              		 	   if(boolobj[this.id] || !bool) {
              		 		   	$.fn.validatebox.defaults.rules.checkNumber.message=ZhiBiaoMenXianHuChi;
                		 		 return false;
              		 	   }
              			 } 
        			threadFormData=JSON.stringify(threadFormData);
        		 }else{
        			 threadFormData=null;
        		 }
               if(param){
            	   threadFormData=JSON.parse(threadFormData);
            	   $.fn.validatebox.defaults.rules.checkNumber.message=ZhiBiaoMenXianShuZhiSheZhiBuHeLi;
                   var begin=$("input[name='"+param[0]+"']").val();
                   if(parseFloat(threadFormData.generalEnd)==parseFloat(threadFormData.seriousEnd)){
                	   return false;
                   }
                   if(begin!=""){
                	   return /^\d+(\.\d+)?$/.test(value)&&(parseFloat(value) > parseFloat(begin));
                   }
                   return true;
               }else{
            	   threadFormData=JSON.parse(threadFormData);
            	   $.fn.validatebox.defaults.rules.checkNumber.message=ZhiBiaoMenXianShuZhiSheZhiBuHeLi;
                   var end=$("input[name='"+this.name.replace('Begin','End')+"']").val();
                   if(parseFloat(threadFormData.generalBegin)==parseFloat(threadFormData.seriousBegin)){
                	   return false;
                   }
                   if(end!=""){
                	   return /^\d+(\.\d+)?$/.test(value)&&(parseFloat(value) < parseFloat(end));
                   }
                   return /^\d+(\.\d+)?$/.test(value);
               }
           }
       }
   })
   
	$(function () {
		closeLoading();
		//初始化或切换 4G,5G时需刷新左侧指标功能集及右侧指标列
		$("#kpimgt_serach_text").val(''); //清空搜索框
		kpiManasearchText = '';

	    if(kpiManagePageVue.currentKpiNetType == 'enb'){
    	    curEnbOrGnbTreeUrl = '${ctx}/pm/indicatormg/getIndicatorGroupTree.action';
		    //eNBOrGnbTree();
        }else if(kpiManagePageVue.currentKpiNetType == 'gnb'){
			curEnbOrGnbTreeUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorGroupTree.action';
			//eNBOrGnbTree();
		}else if(kpiManagePageVue.currentKpiNetType == 'egw'){
			curEnbOrGnbTreeUrl = '${ctx}/egw/pm/indicatormg/getIndicatorGroupTree.action';
		}
		eNBOrGnbTree();
      
       //指标功能集树加载
       //var rootId = "";
       function eNBOrGnbTree(){
    	   setTimeout(function(){
	    	   $("#kpiArithmeticTree").tree({
		           url: curEnbOrGnbTreeUrl,
		           animate: true,
		           lines: true, // 是否显示树形结构中的树线
		           border: true,
		           onBeforeLoad :function(node,param){
		        	   if(kpiManasearchText.length == ""){
			        	   param["noKpiShowGroup"]=1;
		        	   }else{
			        	   param["noKpiShowGroup"]=0;
		        	   }
		        	   param["searchText"]= kpiManasearchText;
		        	   
		        	   if(kpiManagePageVue.currentKpiNetType == 'enb'){
		        		   param["product_type"]= tableProductTypeSelect;
						   //level
							param.indicatorLevel = kpiManagePageVue.levelSelectParam;
		        	   }else if(kpiManagePageVue.currentKpiNetType == 'gnb'){
		        		   //5g 参数 20220812: 暂时将 5g,egw 网元下涉及的产品类型参数隐藏；
		        		   //param["product_type"]= 'BBU_XSS';
		        	   }else{
		        		   //eGW 参数  53047 注意这里的产品类型
		        		   //param["product_type"]= 'WCG';
		        	   }
		        	  
		        	   param["timeZone"]= timeZone;
		           },
		           onLoadSuccess:function(node,data){
		        	   
		               $("#kpiArithmeticTree li:eq(1)").find("div").addClass("tree-node-selected");
		               var n = $("#kpiArithmeticTree").tree("getSelected");  
		              
		               if(n!=null){ 
		                    $("#kpiArithmeticTree").tree("select",n.target);    
		               }else{
		            	   $('#kpiDatagrid').datagrid('loadData',[]);
		            	   //筛选的传参
		                   globalQueryParams = {
		                   		catagoryId: "",
		    		   			searchText: kpiManasearchText,
		    		   			timeZone: timeZone,
		                   };
		                   if(kpiManagePageVue.currentKpiNetType == 'enb'){
		                	    globalQueryParams.product_type = tableProductTypeSelect;
							    //level
								globalQueryParams.indicatorLevel = kpiManagePageVue.levelSelectParam;
			        	   }else if(kpiManagePageVue.currentKpiNetType == 'gnb'){
			        		   //5g 参数
			        		   //globalQueryParams.product_type= 'BBU_XSS';
			        	   }else{
			        		   //eGW 参数
			        		   //globalQueryParams.product_type= 'WCG';
			        	   }
		               }
		            },
		           onBeforeSelect:function(node){
		        	   if(!$(this).tree("isLeaf",node.target)){
		        		   return false;
		        	   } 
		           },
		           onSelect: function (node){
						var curUrl = '',
			        	    params = {
								catagoryId: node.id,
								searchText: kpiManasearchText,
								timeZone: timeZone,
							};
						//置空已选的右侧数据
						//kpiManagePageVue.selectedRows = [];
						//$('#kpiDatagrid').datagrid('clearChecked');
		        	    //筛选的传参
		                globalQueryParams = {
		               		catagoryId: node.id,
				   			searchText: kpiManasearchText,
				   			timeZone: timeZone,
		                };
		                kpiManagePageVue.selectKpiName = node.text;
		        	    
						if(kpiManagePageVue.currentKpiNetType == 'enb'){
							curUrl = '${ctx}/pm/indicatormg/getIndicatorListByPage.action';
	            		    params.product_type = tableProductTypeSelect;
	            		    globalQueryParams.product_type = tableProductTypeSelect;

							//level
							params.indicatorLevel = kpiManagePageVue.levelSelectParam;
							globalQueryParams.indicatorLevel = kpiManagePageVue.levelSelectParam;
	            	    }else if(kpiManagePageVue.currentKpiNetType == 'gnb'){
	            		    curUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorListPageData.action';
	            		    //params.product_type = 'BBU_XSS';
	            		    //globalQueryParams.product_type = 'BBU_XSS';
	            	    }else{
	            	    	 curUrl = '${ctx}/egw/pm/indicatormg/getIndicatorListPageData.action';
		            		 //params.product_type = 'WCG';
		            		 //globalQueryParams.product_type = 'WCG';
	            	    }

		                if(node.id != rootId){
		                	$('#kpiDatagrid').datagrid({
			                    queryParams: params,
			                    url: curUrl
			                });
		                };		              
		           },
		           onContextMenu:function(e){
		              e.preventDefault();
		           },
		           formatter : operateKpiTree
		      }); 
    	   },200);

		   try{
			$('#kpiManagePage table.datagrid-f').datagrid('resize');
			//$('table.datagrid-f').datagrid('resize');
			  window.dispatchEvent(new Event('resize'));
		  }	catch(e){}
       };

      	//产品类型下拉选择
	   if(kpiManagePageVue.currentKpiNetType == 'enb'){
		   axios.post('${ctx}/cell/cpeinfos/getEnbMonitorProductList.action?isGnb=0&isAll=1').then(function(response){
		   		var data = response.data;
		   		// 动态删除 BTS
		   		data = data.filter(function(item) {
		   			return item !== 'BTS';
		   		});
		   		newData = data;
		   		if(data.length == 0){
		   			tableProductTypeSelect = 'no';
		   		}else{
		   			//tableCurProduct = data.join(',');
			   		tableCurProduct = 'ALL,' + data.join(',');
			   		tableProductTypeSelect = tableCurProduct;
		   		}
		   		var arr = [{name:'<%=rb.getString("QuanBu")%>',value:''}];
		   		data.map(function(item){
		   			if (item){
		   				arr.push({name:item,value:item})
		   			}
		   		})
		   		kpiManagePageVue.productOptions = arr;
		   		
		   }).catch(function(error){})
	   }else if(kpiManagePageVue.currentKpiNetType == 'gnb'){
			//5g 参数
	        //tableProductTypeSelect = 'BBU_XSS';
	   }else{
		   //eGW 参数
	        //tableProductTypeSelect = 'WCG';
	   }
	  
       //超级管理员，基础指标新建可见
       //回车事件
       $("#kpimgt_serach_text").bind("keyup", function(e){
           if (e.keyCode == 13){
               kpiArithmeticTreeLoad();
           }
       });
	   
		var kpiDatagrid_columns = [
    	   	{
				field : 'ck',
				checkbox:true,
			},
			{
               field : 'operation',
               title : '',
               fixed : true,
               width : 85,
               hidden : false,
               formatter:function(value,rowData,rowIndex){
                   var rowId = rowData.kpiId,
                   	   isCustomize = rowData.isCustomize==1;
                   var value = "<div class='el-icon el-icon-operation-info defaultIconColor' title='"+ChaKan+"' onclick='viewCustomizingKPI("+isCustomize+")'></div>"
                  
					if(rowData.isCustomize == '1'){
						if(rowData.indicatorType == 'counter') {
							value += "<div class='el-icon el-icon-operation-edit CODE_PERFORMANCE_MANAGEMENT hidden' style='margin-left:10px;background-position-x:0;' title='"+XiuGai+"' onclick='viewCustomizingKPI(false,true)'></div>"
						}else {
							value += "<div class='el-icon el-icon-operation-edit CODE_PERFORMANCE_MANAGEMENT hidden' style='margin-left:10px;background-position-x:0;' title='"+XiuGai+"' onclick='modifyCustomizingKPI()'></div>"
						}
						value += '<div class="el-icon el-icon-operation-delete CODE_PERFORMANCE_MANAGEMENT hidden" style="margin-left:10px;background-position-x:0;" title="'+ShanChu+'" onclick="deleteCustomizingKPI(\''+ rowId +'\')"></div>';
					}else {
						value += "<div class='el-icon el-icon-operation-edit CODE_PERFORMANCE_MANAGEMENT hidden' style='margin-left:10px;background-position-x:0;' title='"+XiuGai+"' onclick='viewCustomizingKPI(false,true)'></div>"
					}
                   
                   return value;
               },
           },
    	   {
               field : 'isEnable',
   			   sortable : true,
               title : '<%=rb.getString("CeLiangNew")%>',
               width : 50,
               formatter:function(value,rowData,rowIndex){
            	   if(value == '0'){
                	   value = '<%=rb.getString("Fou")%>'
                   }else if (value == '1'){
                	   value = '<%=rb.getString("Shi")%>'
                   }else{
                	   value = ''
                   }
                   return value;
               },
           },
    	   {
               field : 'kpiId',
   			   sortable : true,
               title : 'Counter/<%=rb.getString("ZhiBiaoID")%>',
               width :100
           }, {
               field : 'kpiName',
   			   sortable : true,
               title : 'Counter/<%=rb.getString("ZhiBiaoMingCheng")%>',
               width : 140,
			   formatter: function(value,row,rowIndex) {
					var str = '';

					if([true, 'true'].includes(row.isConflict)) {
						str = '<span class="el-icon el-icon-circle-warning" onclick="duplicateWarningClick(&quot;' + row.kpiId + '&quot;,&quot;' + value + '&quot;)" style="color: #FFAA00;"></span> ';
					}

					str += value;

					return str;
			   }
           },
           {
               field : 'product_type',
   			   sortable : false,
               title : '<%=rb.getString("ChanPinLeiXing")%>',
               width : 90
              
           },
		   {
			   field : 'custName',
			   sortable : true,
			   title : '<%=rb.getString("ZiDingYiZhiBiaoMingChen")%>',
			   width : 140
		   },
		    //level 字段 仅在 enb 页面显示
		  	{
				field : 'indicatorLevel',
                title : '<%=rb.getString("DengJi")%>',
                width : 40,
				formatter:function(value,rowData,rowIndex){
                   if(value == 'plmn'){
                	   value = 'PLMN'
                   }else{
                	   value = 'Device'
                   }
                   return value;
               },
           	},
		   {
			   field : 'unit',
               title : '<%=rb.getString("DanWei")%>',
               width : 50
           }, {
        	   field : 'isCustomize',
   			   sortable : true,
               width : 80,
               formatter:function(value,rowData,rowIndex){
                   if(value == '0'){
                	   value = '<%=rb.getString("JiChuZhiBiao")%>'
                   }else{
                	   value = '<%=rb.getString("ZiDingYiZhiBiao")%>'
                   }
                   return value;
               },
               title : titleFilter
           },          
           {
               field : 'updater',
   			   sortable : true,
               title : '<%=rb.getString("GengXinRen")%>',
               width : 50
           },{
               field : 'updateTime',
   			   sortable : true,
               title : '<%=rb.getString("GengXinShiJian")%>',
               width : 80
           }
       	];

		
       //基础指标 datagrid
       $('#kpiDatagrid').datagrid({
   		   idField:'kpiId',
           border: false,
           fit:true,
           cellpadding: false,
           nowrap : false,
           striped : true,
           singleSelect:singleArr[0],
           rownumbers: true, 
           fitColumns:true,
           pagination: true, 
           checkOnSelect: false,
           selectOnCheck: true,
           pagePosition: 'bottom',
           toolbar:'#kpiDatagridToolbar',
           onLoadError: datagridLoadError,
           onBeforeLoad : beforeLoad_kpiDatagrid,
           onLoadSuccess: gridOnLoadSuccessForAutoSize,
           onSelect: gridOnSelectForReloadView,
           columns : [kpiDatagrid_columns],
           onCheck:checkEnable,
           onUncheck:uncheckEnable,
           onCheckAll:checkEnableAll,
           onUncheckAll:uncheckEnableAll,
           onBeforeSelect: onBeforeSelectEnable,
           onClickRow: function(rowIndex,rowData){
	        	curOnClickRowId =  rowData.kpiId;
	       }
       });
      
   	//64114 接口每隔6秒更新一次
	/*if(updateTableDataTimer){
		clearInterval(updateTableDataTimer);
	}
	updateTableDataTimer = setInterval(function(){    
		var kpiManagePageCtn = $("#kpiManagePage");			
		if(!kpiManagePageCtn.length) {
			clearInterval(updateTableDataTimer);
			return;
		}
		
		$('#kpiDatagrid').datagrid('reload');
	},6000);*/
       /* 权限控制 */
       setTimeout(function(){
	       $('#kpiManageTabsDiv span[tabtit]:visible').each(function(n,item){
	   			if(n==0) $(item).click();
	   	   });
	   },0);
	   /* 菜单隐藏处理 */
	   $('#omc_app_ctn').mousedown(function(){
			try{
			    var target = event.target, list = Array.from(target.classList),
			        plist = Array.from(target.parentNode.classList);
			    if(!(list.includes('filter-menu') || list.includes('filter-item') || plist.includes('filter-item'))){
			      $('.filter-menu').hide();
			    }
			}catch(e){}
		});
    });
   var cus = false;

    /* counter 重复提醒点击事件 */
	function duplicateWarningClick(id, name) {
		var vm = kpiManagePageVue;

		Object.assign(vm.duplicateForm, {
			kpiId: id,
			indicatorName: name
		});
		vm.openDuplicateDl();
	}
   
   	/* 表格复选框选中事件 */
	function checkEnable(index,row){
    	var rows = $('#kpiDatagrid').datagrid('getChecked');
    	kpiManagePageVue.selectedRows = rows;
   	}
   	/* 表格复选框取消选中事件 */
   	function uncheckEnable(index,row){
   		var rows = $('#kpiDatagrid').datagrid('getChecked');
   		kpiManagePageVue.selectedRows = rows;   		
    }
    /* 表格复选框选中所有事件 */
	function checkEnableAll(rows){
   		$.each(rows,function(index,item){
   			checkEnable('',item);
   		})
   	}

    /* 表格复选框取消选中所有事件 */
    function uncheckEnableAll(rows){
   		$.each(rows,function(index,item){
   			uncheckEnable('',item);
   		})
    }
    //选中指标，点击测量
  	function enableBulkClick(){
		var curCheckList = $('#kpiDatagrid').datagrid('getChecked');
		if(curCheckList.length > 0){
			var curIndicatorIds = curCheckList.map(function(item){ return item.kpiId}).join(',');	
		}else{
			return
		}
  		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingQiYong")%>", function (r) {
	        if (r) {
	            var params = {
	            	indicatorIds : curIndicatorIds
	            };
	            $.post("${ctx}/cell/perfmgmt/kpimanage/enableIndicator.action", params, function (data) {
	            	if(data["success"]){
	            		cancleEnbaleClick();
	            	} else {
 	                	showMsg('error_msg',data["msg"]);
 	                }
	            }, "json");
	        }
	    }).addClass("seriousConfirm");
   	}
    //选中指标，点击取消测量
	function disableBulkClick(){
		var curCheckList = $('#kpiDatagrid').datagrid('getChecked');
		
		if(curCheckList.length > 0){
			var curIndicatorIds = curCheckList.map(function(item){ return item.kpiId}).join(',');
            var params = {
                indicatorIds: curIndicatorIds
            };
		}else{
			return;
		}
		//查询指标是否已被关联到模板 1-已被模板关联，0-未被模板关联
		axios.get('${ctx}/cell/perfmgmt/kpimanage/isIndicatorInTemplate.action',{ params: { indicatorIds: curIndicatorIds}}).then(function(response){
            var data = response.data;

            if(data){
                 var combArr = Object.keys(data).map((el) => {
                    return {
                        kpiId: el,
                        result: data[el]
                    }
                })
				
                kpiManagePageVue.kpiTableData = combArr.filter((val) => val.result !== '0');
            }else{
                kpiManagePageVue.kpiTableData = [];
            }
        }).catch(function(error){})

		kpiManagePageVue.kpiResultTipDialog = true;
		kpiManagePageVue.operOrMainPageFlag = true;
	}

	function cancleEnbaleClick(){
		$('#kpiDatagrid').datagrid('reload');
		$('#kpiDatagrid').datagrid('clearChecked')
	}
	/**
	* 表格数据 行点击事件
	* @param index{number}   下标
	* @param row{object}   行数据
	*/
	function onBeforeSelectEnable(index,row){
		 var tb = $(this);
		slData = tb.datagrid('getChecked'),
		rows = tb.datagrid('getRows');
		$.each(rows,function(index,item){
			if($.inArray(item,slData) < 0 && item!=row) {
				var rowIndex = tb.datagrid('getRowIndex',item);
				tb.datagrid('unselectRow',rowIndex);
			}    
		});
	}
   /* 选中自动更新查看页面 */
   function gridOnSelectForReloadView(index,row){
	   var slide = $("#kpiManaAddOrModify"),
	   	   right = slide.css('right').replace('px','');
	   /* 侧划是可见的 */
	   if(right>=0){
		   try{
			   var opts = slide.panel("options");
			   /* 确定为查看页面时刷新 */
			   if(opts.href.indexOf('viewKPIInfo')>=0){
				   viewCustomizingKPI();
			   }
		   }catch(e){}
	   }
   }
   
	// 基本指标集  树形结构
	function kpiArithmeticTreeLoad(){
		kpiManasearchText = $("#kpimgt_serach_text").val();
		$("#kpiArithmeticTree").tree("reload");

		try{
			$('#kpiManagePage table.datagrid-f').datagrid('resize');
			//$('table.datagrid-f').datagrid('resize');
			window.dispatchEvent(new Event('resize'));
		}	catch(e){}
	}

	// 处理ENB网元下父级节点添加图标点击事件
	function handleEnbParentNodeAdd(nodeId) {
		// 通过 nodeId 从树中获取完整的 node 对象
		var node = $('#kpiArithmeticTree').tree('find', nodeId);
		// 调用现有的ENB指标功能集添加功能
		try {
			if(node){
				if(node.device_type == 'ENB'){
					kpiManagePageVue.goAddKpiGroup('ENB');
				}else{
					kpiManagePageVue.goAddKpiGroup('GSM');
				}
			}
		} catch(error) {
		}
	}
   
   function operateKpiTree(node){
	   if(node.children && node.children.length > 0 ){
		   // 只在ENB网元下，为第一层父节点添加添加图标
		   // 使用 netType 字段来判断节点是否应该显示添加图标
		   if(kpiManagePageVue.currentKpiNetType === 'enb'){
			   // 检查节点是否有 netType 字段，且值为 'enb' 或 'gsm'
			   // 这样只有标记了网元类型的第一层父节点才会显示图标
			   if(node.device_type && (node.device_type === 'ENB' || node.device_type === 'GSM')){
				//console.log(node,'....node in formatter')
				   return node.text + 
					   "<div class='operationDiv el-icon el-icon-plus' " +
					   "title='添加指标功能集' onclick='handleEnbParentNodeAdd(\"" + node.id + "\"); event.stopPropagation();' " +
					   "style='float:right; margin-right:5px; margin-top:8px; cursor: pointer; color: #7A7992; font-size: 14px;'></div>";
			   }
		   }
		   return node.text ;
	   }else{
		   var nodeTxt = node.text;
		   if(nodeTxt.length>21) nodeTxt = nodeTxt.substring(0,20)+'...';
		   var textString = "<span title='"+node.text+"'>"+nodeTxt+"</span>" ;
		   if(node.attributes){
				if(node.attributes.isCustomize == '1'){
					textString += "<div style='float:right;margin-right:0px;margin-left:5px;background-position-x:0;font-size:16px; margin-top:5px;' class='operationDiv el-icon el-icon-operation-delete  CODE_PERFORMANCE_MANAGEMENT hidden' title='"+ ShanChu+"' onclick='deleteKpiGroup()'>"
								+ "</div><div style='float:right;background-position-x:0;font-size:16px;margin-top:5px;' class='operationDiv el-icon el-icon-operation-edit  CODE_PERFORMANCE_MANAGEMENT hidden' title='"+XiuGai+"' onclick='goModifyKpiGroup(true,true)'></div>";
				} 
		   }
		   return textString;					
	   }
   }
   
    function returnTextStyleFun(node){
	    var nodeTxt = node.text;
	    if(node.id == "root"){
		    return node.text
	    }else{
		    if(nodeTxt.length>16) nodeTxt = nodeTxt.substring(0,20)+'...';
		    return "<span title='"+node.text+"'>"+nodeTxt+"</span>"
	    }
    }
    // 新建，修改  初始化指标功能集下拉列表
	function initFuncSetValue(id,defaultValue){
    	var curIndicatorGroupUrl = '';
    	
    	if(kpiManagePageVue.currentKpiNetType == 'enb'){
			curIndicatorGroupUrl = '${ctx}/pm/indicatormg/getIndicatorGroupTree.action';
    	}else if(kpiManagePageVue.currentKpiNetType == 'gnb'){
    		curIndicatorGroupUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorGroupList.action';
    	}else{
    		curIndicatorGroupUrl = '${ctx}/egw/pm/indicatormg/getIndicatorGroupList.action';
    	}
    	
		if(kpiManagePageVue.currentKpiNetType == 'enb'){
			$('#'+id).combotree({
				url: curIndicatorGroupUrl,
				panelWidth: 200,
				editable: false,
			onLoadSuccess: function(node, data){
				// 如果有默认值，使用默认值
				if(defaultValue){
					$('#'+id).combotree("setValue", defaultValue);
					// 立即设置 device_type，用于过滤 kpiSetTree
					var tree = $('#'+id).combotree('tree');
					var selectedNode = tree.tree('find', defaultValue);
					if(selectedNode && typeof selectedFuncSetDeviceType !== 'undefined'){
						selectedFuncSetDeviceType = selectedNode.device_type || '';
						console.log('默认值的 device_type:', selectedFuncSetDeviceType);
					}
					// 延迟更新显示文本
					setTimeout(function(){
						if(selectedNode){
							var parentNode = tree.tree('getParent', selectedNode.target);
							if(parentNode){
								$('#'+id).combotree('setText', parentNode.text + '/' + selectedNode.text);
							}
						}
					}, 100);
				} else {
					// 否则默认选中第一个父节点下的第一个子节点
					if(data && data.length > 0 && data[0].children && data[0].children.length > 0){
						var firstChild = data[0].children[0];
						$('#'+id).combotree("setValue", firstChild.id);
						// 设置显示文本为 父级/子级
						$('#'+id).combotree('setText', data[0].text + '/' + firstChild.text);
						console.log('默认选中第一个子节点:', firstChild.text, firstChild.id);
						
						// 立即设置 device_type，用于过滤 kpiSetTree
						if(typeof selectedFuncSetDeviceType !== 'undefined'){
							selectedFuncSetDeviceType = firstChild.device_type || '';
							console.log('默认选中节点的 device_type:', selectedFuncSetDeviceType);
						}
					}
				}
			},			});
		}else{
			//5g 和 egw 指标功能集，保持原样
			$('#'+id).combobox({
     	  			url:curIndicatorGroupUrl,
				valueField:'catagoryId',    
				textField:'catagoryName',
				panelHeight:'auto',
				editable:false,
				onLoadSuccess:function(data){
					if(defaultValue){
						$('#'+id).combobox("setValue",defaultValue);
					}else{
						$('#'+id).combobox("setValue",data[0].catagoryId);
					}
				}
			});
		}
     }
	// 新建，修改初始化指标单位下拉列表
    function initKPIUnit(id,defaultValue){
    	var curIndicatorUnitUrl = '';
    	
    	if(kpiManagePageVue.currentKpiNetType == 'enb'){
    		curIndicatorUnitUrl = '${ctx}/pm/indicatormg/getIndicatorUnitList.action';
    	}else if(kpiManagePageVue.currentKpiNetType == 'gnb'){
    		curIndicatorUnitUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorUnitList.action';
    	}else{
    		curIndicatorUnitUrl = '${ctx}/egw/pm/indicatormg/getIndicatorUnitList.action';
    	}
    	
     	$('#'+id).combobox({    
     	    url : curIndicatorUnitUrl,    
     	    valueField:'code',    
     	    textField:'name',
     	    panelHeight:'auto',
         	editable:false,
         	onChange:function(newValue){
         		if (newValue && newValue != ""){
         			$("#kpiThresholdLabel").text("<%=rb.getString("ZhiBiaoMenXianFanWei")%>("+ $(this).combobox("getText") +")");
         			$("#kpiIdUnitValueIdTittle").html("");
         		}
         	},
         	onLoadSuccess:function(data){
         		if(defaultValue){
	         		$('#'+id).combobox("setValue",defaultValue);
         		}else{
	         		$('#'+id).combobox("setValue",data[0].code);
         		}
            }
     	});
     }
     
     
     //初始化统计类型下拉列表 
     function initStatisticSetValue(id,defaultValue){
      	$('#'+id).combobox({
      	    valueField:'statisticId',    
      	    textField:'statisticName',
      	    panelHeight:'auto',
          	editable:false,
          	data:[
          		{
          			statisticId:'sum',
          			statisticName:'Sum'
          		},
          		{
          			statisticId:'avg',
          			statisticName:'Avg'
				},{
          			statisticId:'pct',
          			statisticName:'Pct'
          		},{
          			statisticId:'max',
          			statisticName:'Max'
          		}
          	],
          	onLoadSuccess:function(data){
         		if(defaultValue){
	         		$('#'+id).combobox("setValue",defaultValue);
         		}else{
	         		$('#'+id).combobox("setValue",data[0].statisticId);
         		}
            }
      	});
      }
  	 //初始化统计类型下拉列表 
     function initenableSetValue(id,defaultValue,flagType){
      	$('#'+id).combobox({
      	    valueField:'id',    
      	    textField:'text',
      	    panelHeight:'auto',
          	editable:false,
			value: '0',
          	data:[
          		{
          			id:'0',
          			text:'<%=rb.getString("Fou")%>'
          		},
          		{
          			id:'1',
          			text:'<%=rb.getString("Shi")%>'
				}
          	],
          	onLoadSuccess:function(data){
         		if(defaultValue){
	         		$('#'+id).combobox("setValue",defaultValue);
         		}else{
	         		$('#'+id).combobox("setValue",data[0].id);
         		}
            },
            //只有 enb 才有查询关联模板  为 No 查询指标是否已被关联到模板 1-已被模板关联，0-未被模板关联，只限于是修改
            onChange: function(newValue,oldValue){
                if(newValue == '0' && modifyOrViewKpiFlag == true && flagType != 'add'){
                    if(curOnClickRowId){
                        //查询指标是否已被关联到模板 1-已被模板关联，0-未被模板关联
                        axios.get('${ctx}/cell/perfmgmt/kpimanage/isIndicatorInTemplate.action',{ params: { indicatorIds: curOnClickRowId}}).then(function(response){
                        var data = response.data;
                       
                            if(data){
                                 var combArr = Object.keys(data).map((el) => {
                                     return {
                                         kpiId: el,
                                         result: data[el]
                                     }
                                })
                                kpiManagePageVue.kpiTableData = combArr.filter((val) => val.result !== '0');

                                if(kpiManagePageVue.kpiTableData.length > 0){
                                    kpiManagePageVue.kpiResultTipDialog = true;
                                    kpiManagePageVue.operOrMainPageFlag = false;
                                }
                            }else{
                                kpiManagePageVue.kpiTableData = [];
                                kpiManagePageVue.kpiResultTipDialog = false;
                                kpiManagePageVue.operOrMainPageFlag = false;
                            }
                        }).catch(function(error){})

                    }else{
                        return;
                    }

                }else{
                    kpiManagePageVue.kpiResultTipDialog = false;
                    kpiManagePageVue.operOrMainPageFlag = false;
                }
            }
      	});
      }
          
    // 加载功能集
	function loadFuncSet(type){
		var columnsList = [];
		
    	if(kpiManagePageVue.currentKpiNetType == 'enb'){
			columnsList = [
				{field : 'isEnable', sortable : true,  title : '<%=rb.getString("CeLiangNew")%>', width : 100,
		              formatter:function(value,rowData,rowIndex){
		                   if(value == '0'){
		                	   value = '<%=rb.getString("Fou")%>'
		                   }else if (value == '1'){
		                	   value = '<%=rb.getString("Shi")%>'
		                   }else{
		                	   value = ''
		                   }
		                   return value;
		               },
		         },
            	 {field : 'kpiId',sortable : true,title : '<%=rb.getString("ZhiBiaoID")%>',width : 150}, 
                 {field : 'kpiName',sortable : true,title : '<%=rb.getString("ZhiBiaoMingCheng")%>',width : 200},
                 {field : 'custName', title : '<%=rb.getString("ZiDingYiZhiBiaoMingChen")%>', width : 200},
                 {field : 'unit',title : '<%=rb.getString("DanWei")%>',width : 90},
                 {field : 'product_type',title : '<%=rb.getString("ChanPinLeiXing")%>',width : 150}
             ] 
		}else if(kpiManagePageVue.currentKpiNetType == 'gnb'){			
			columnsList = [	 
            	 {field : 'kpiId', sortable : true, title : '<%=rb.getString("ZhiBiaoID")%>',width : 280}, 
            	 {field : 'kpiName',sortable : true,title : '<%=rb.getString("ZhiBiaoMingCheng")%>',width : 250},
            	 {field : 'custName', title : '<%=rb.getString("ZiDingYiZhiBiaoMingChen")%>', width : 250},
            	 {field : 'unit',title : '<%=rb.getString("DanWei")%>',width : 200}
             ] 
		}else{
			columnsList = [	 
           		{field : 'kpiId', sortable : true, title : '<%=rb.getString("ZhiBiaoID")%>',width : 280}, 
           		{field : 'kpiName',sortable : true,title : '<%=rb.getString("ZhiBiaoMingCheng")%>',width : 250},
           		{field : 'custName', title : '<%=rb.getString("ZiDingYiZhiBiaoMingChen")%>', width : 250},
           		{field : 'unit',title : '<%=rb.getString("DanWei")%>',width : 200}
            ]
		}
    	
         // 新建修改自定义指标-窗口-指标列表
         $('#kpiListDatagrid').datagrid({
        	 checkbox:true,
             border: false,
             fit: true,
             rownumbers:true, 
             fitColumns : true,
             nowrap : false,
             striped : true,
             singleSelect: true,
             pagination:true, 
             pagePosition: 'bottom',
             onLoadError:datagridLoadError,
             onLoadSuccess:datagridLoadSuccess,
             columns : [columnsList],
             onClickCell: function(rowIndex, field, value) {
            	 $("#calcExpValueShowTitle").html("");
            	 var grid = $('#kpiListDatagrid');
                 var selOperator = $('#kpiDatagrid').datagrid('getSelected');
                 var oldVal = $('#calcExpValue').val().replace(/\n/g, " ");
                 var kpiArith = grid.datagrid('getRows')[rowIndex].kpiId;
                 $('#calcExpValue').val(oldVal+kpiArith);
                 
                 var oldShowVal = $('#calcExpValueShow').val().replace(/\n/g, " ");
                 var kpiArithName = "",
                 	 isCustomizing;
                 
                 isCustomizing = grid.datagrid('getRows')[rowIndex].isCustomize;
                 kpiArithName = grid.datagrid('getRows')[rowIndex].kpiId;
                 var kpiText = grid.datagrid('getRows')[rowIndex].kpiName;
                 
                 //4g有， 5g，egw无 product_type
                 if(kpiManagePageVue.currentKpiNetType == 'enb'){
                 	var commonProduct = grid.datagrid('getRows')[rowIndex].product_type.replace(/\//g,','); //产品类型
                     params = {key:kpiArith,value:kpiArithName,name:kpiText, product_type: commonProduct};
                 }else{                	
                     params = {key:kpiArith,value:kpiArithName,name:kpiText};
                 }
                 addRuleDom('#calcExpValueShow',params,updateCalcExpValue);
                 
                 if($("#calcExpValueShow").val().length>0){
             		$("#calcExpValueShowTitle").html("");
             	}
             }
         });
         
     	// kpi指标功能集算法树形结构 新增修改页面 
     	reloadKpiTree(type);
     }

   
     function reloadKpiTree(type){
    	searchTextKPIZhiBiao = $("#kpi_search_text").val();//新建页面- kpi 搜索条件
    	var curTreeUrl = '', curkpiListUrl = '';
    	
    	if(kpiManagePageVue.currentKpiNetType == 'enb'){
    		curTreeUrl = '${ctx}/pm/indicatormg/getIndicatorGroupTree.action?noKpiShowGroup=0';
    		curkpiListUrl = '${ctx}/pm/indicatormg/getIndicatorListByPage.action';
    	}else if(kpiManagePageVue.currentKpiNetType == 'gnb'){
    		curTreeUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorGroupTree.action?noKpiShowGroup=0';
    		curkpiListUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorListPageData.action';
    	}else{
    		curTreeUrl = '${ctx}/egw/pm/indicatormg/getIndicatorGroupTree.action?noKpiShowGroup=0';
    		curkpiListUrl = '${ctx}/egw/pm/indicatormg/getIndicatorListPageData.action';
    	}
    	
    	$("#kpiSetTree").tree({
			url: curTreeUrl,
            animate: true,
            lines: true, // 是否显示树形结构中的树线
            border: true,
            onBeforeLoad:function(node,param){		            	 
				param["noKpiShowGroup"]=0;
  	        	param["searchText"]=searchTextKPIZhiBiao;
  	        	if(kpiManagePageVue.currentKpiNetType == 'enb'){
  	        		if(newData.length == 0){
	  	        		  param["product_type"] = 'no';
	  	        	}else{
	  	        		  param["product_type"] = searchProductTypeSelect;
	  	        	}

					//level 不需要该参数
					//param.indicatorLevel = kpiManagePageVue.levelSelectParam;
	  	       	}else if(kpiManagePageVue.currentKpiNetType == 'gnb'){
	  	       	 	//param["product_type"]='BBU_XSS';
	  	       	} else{
	  	       		//param["product_type"]='WCG';
	  	       	}
            },
            onLoadSuccess:function(node,data){
                 // 根据选中功能集的 device_type 过滤树形数据
                 if(typeof selectedFuncSetDeviceType !== 'undefined' && selectedFuncSetDeviceType && data){
                     var filteredData = data.filter(function(item){
                         return item.device_type === selectedFuncSetDeviceType;
                     });
                     console.log('过滤后的树形数据 (device_type=' + selectedFuncSetDeviceType + '):', filteredData);
                     // 重新加载过滤后的数据
                     $("#kpiSetTree").tree("loadData", filteredData);
                     return;
                 }
                 
                 $("#kpiSetTree li:eq(1)").find("div").addClass("tree-node-selected");
                 var selKpiId = "";
                 if($('#kpiDatagrid').datagrid('getSelected')){
        	 	 	  selKpiId = $('#kpiDatagrid').datagrid('getSelected').kpiId;
                 }
	             var n = $("#kpiSetTree").tree("getSelected");  
                 if(n!=null){ 
                     $("#kpiSetTree").tree("select",n.target);
                 }else{
                 	$('#kpiListDatagrid').datagrid("loadData",[]);
                 }
            },
            onBeforeSelect:function(node){
            	 if(!$(this).tree("isLeaf",node.target)){
            		  return false;
            	 } 
            },
            onSelect: function (node){
				var selKpiId = "";
                if($('#kpiDatagrid').datagrid('getSelected')){
        	 	 	 selKpiId = $('#kpiDatagrid').datagrid('getSelected').kpiId;
                }
             	var params = { catagoryId: node.id,noKpiIds:selKpiId,searchText:searchTextKPIZhiBiao, isEnable: '1'};
             	if(kpiManagePageVue.currentKpiNetType == 'enb'){
             		params["product_type"] = searchProductTypeSelect;
					//level
					//params.indicatorLevel = kpiManagePageVue.levelSelectParam;
					//应该是新建指标的等级取值
					if(type == 'add'){
						//新建指标
						addOrModifyKpiLevel = $('#kpiManaAdd_body [name=indicatorLevel]:checked').val();
					}else{
						//修改指标
						addOrModifyKpiLevel = $('#kpiManaModify_body [name=modifyLevelType]:checked').val();
						
					}
					params.indicatorLevel = addOrModifyKpiLevel;		
					
          	    }else if(kpiManagePageVue.currentKpiNetType == 'gnb'){
          		     //5g 参数
          		     //params["product_type"] = 'BBU_XSS';	          		    
          	    }else{
          	    	 //eGW 参数
					// params["product_type"] = 'WCG';
          	    }
             	if(type == 'add')	delete params.noKpiIds;
             	$('#kpiListDatagrid').datagrid({
             		url : curkpiListUrl,
             		queryParams: params,
             		pageNumber : 1
             	 });
             },
             formatter : returnTextStyleFun
         });		
     
		 try{
			$('#kpiManagePage table.datagrid-f').datagrid('resize');
			//$('table.datagrid-f').datagrid('resize');
			window.dispatchEvent(new Event('resize'));
		}	catch(e){}
	}
     //右上角 新增kpi 指标页面
     function goAddKpiArithmeticWin(){
    	 $("#kpiManaAdd").slideDown(500,function(){
    		 $('#kpiManaAdd').addClass('loading');
   			 $("#kpiManaAdd").load('${ctx}/cell/perfmgmt/kpimanage/goKPIManageWindPage.action?code=arithmetic&randomValue='+strRandom,function(data){				
   				$.parser.parse(this);
   				$('#kpiManaAdd').removeClass('loading');
   			});
   		 });
     }
     
     // 查看或修改基础指标 KPI
     function viewCustomizingKPI(isCustom,bool) {
    	 var suffix = '';
    	 if(bool == true) {
    	    suffix += '?isBasic=true';
    	    //只有 enb 支持 查询关联模板
            if(kpiManagePageVue.currentKpiNetType == 'enb'){
                modifyOrViewKpiFlag = true;
            }else{
                modifyOrViewKpiFlag = false;
            }
    	 }else{
             modifyOrViewKpiFlag = false;
    	 }

    	 if(isCustom == true) {
    		 suffix += suffix? '&':'?';
    		 suffix += 'isCustomView=true';
    	 }
    	 closekpiManaTemplate();
    	 //接口后加参数 isBasic: true 为修改基础指标跳转；
    	 //接口不加参数  为查看基础指标页面
    	 //此接口不区分 4G,5G,eGW
		 $("#kpiManaAddOrModify").animate({right:'0px'},500,function(){
			 $('#kpiManaAddOrModify').addClass('loading');
			 $('#kpiManaAddOrModify').load( '${ctx}/pm/indicatormg/viewKPIInfo.action'+suffix,function(){
				 $.parser.parse(this);
				 $('#kpiManaAddOrModify').removeClass('loading');
			 });
		 });
     }
     // 修改指标 KPI(自定义指标)
    function modifyCustomizingKPI() {
        //只有 enb 支持 查询关联模板
        if(kpiManagePageVue.currentKpiNetType == 'enb'){
            modifyOrViewKpiFlag = true;
        }else{
            modifyOrViewKpiFlag = false;
        }
    	 closekpiManaTemplate();
    	 //此接口不区分 4G,5G,eGW
		 $("#kpiManaAddOrModify").animate({right:'0px'},500,function(){
			 $('#kpiManaAddOrModify').addClass('loading');
			 $('#kpiManaAddOrModify').load( '${ctx}/pm/indicatormg/goModifyKPIPage.action?randomValue='+strRandom,function(){
				 $.parser.parse(this);
				 $('#kpiManaAddOrModify').removeClass('loading');
			 });
		 });
     }
     //删除指标 KPI
     function deleteCustomizingKPI(rowId,rowName){
     	var deleteKpiUrl = '',
     		params = {
     			"kpiId":rowId
     		};
     	if(kpiManagePageVue.currentKpiNetType == 'enb'){
     		deleteKpiUrl = '${ctx}/pm/indicatormg/delIndicator.action';
     	}else if(kpiManagePageVue.currentKpiNetType == 'gnb'){
     		deleteKpiUrl = '${ctx}/gnb/pm/indicatormg/delIndicator.action';
     	}else{
     		deleteKpiUrl = '${ctx}/egw/pm/indicatormg/delIndicator.action';
     	}
     	
     	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuZhiBiao")%>", function (r) {
  	        if (r) {
  	           $.post(deleteKpiUrl, params, function(data) {
  	    			if (data["success"]) {
  	    				cancleEnbaleClick();
  	                } else {
  	                	showMsg('error_msg',data["message"]);
  	                    return;
  	                }
  	            }, "json");
  	        }
  	    }).addClass('seriousConfirm');
     }

   	 //修改指标功能集组  user_code当前用户
     function goModifyKpiGroup() {
		
		setTimeout(function(){
			var curInfoUrl = '', node=$('#kpiArithmeticTree').tree("getSelected");
			console.log(node,'.goModifyKpiGroup....node')

			kpiManagePageVue.curModifyFunNameId = node.id;
	    	kpiManagePageVue.curFunNameType = 'modify';

	    	if(kpiManagePageVue.currentKpiNetType == 'enb'){
				if(node.device_type == 'ENB'){
					kpiManagePageVue.enbOrGnbTitle = '<%=rb.getString("XiuGaiENBZhiBiaoGongNengJi")%>';
					kpiManagePageVue.commonName = '<%=rb.getString("XiuGaiENBZhiBiaoGongNengJi")%>';
				}else if (node.device_type == 'GSM'){
					kpiManagePageVue.enbOrGnbTitle = '<%=rb.getString("XiuGaiGSMZhiBiaoGongNengJi")%>';
					kpiManagePageVue.commonName = '<%=rb.getString("XiuGaiGSMZhiBiaoGongNengJi")%>';
				}

				curInfoUrl = '${ctx}/pm/indicatormg/getIndicatorGroupInfo.action';
			}else if(kpiManagePageVue.currentKpiNetType == 'gnb'){
				kpiManagePageVue.enbOrGnbTitle = '<%=rb.getString("XiuGaiZhiBiaoGongNengJi")%>';
				kpiManagePageVue.commonName = '<%=rb.getString("XiuGaiZhiBiaoGongNengJi")%>';

				curInfoUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorGroupInfo.action';
			}else{
				kpiManagePageVue.enbOrGnbTitle = '<%=rb.getString("XiuGaiZhiBiaoGongNengJi")%>';
				kpiManagePageVue.commonName = '<%=rb.getString("XiuGaiZhiBiaoGongNengJi")%>';

				curInfoUrl = '${ctx}/egw/pm/indicatormg/getIndicatorGroupInfo.action';
			}
			
		   	$.post(curInfoUrl, {timeZone:timeZone,catagoryId:node.id}, function (data) {
		    	if(data){
		    		kpiManagePageVue.addFunNameForm.catagoryName = data.catagoryName || '';
		    		kpiManagePageVue.addFunNameForm.description = data.description || '';
		    	}
			}, "json");
	    	kpiManagePageVue.addFunNameShow = true;

			//表格自适应
			try{
				$('#kpiManagePage table.datagrid-f').datagrid('resize');
				//$('table.datagrid-f').datagrid('resize');
				window.dispatchEvent(new Event('resize'));
			}	catch(e){}
		},300);	
    }
     //删除指标功能集
     function deleteKpiGroup(){
    	 var curDeleteGroupUrl = '';
    	 //关闭右侧新建或修改指标功能集窗口
    	 kpiManagePageVue.addFunNameShow = false;
    	 
    	 if(kpiManagePageVue.currentKpiNetType == 'enb'){
    		 curDeleteGroupUrl = '${ctx}/pm/indicatormg/delIndicatorGroup.action';
    	 }else if(kpiManagePageVue.currentKpiNetType == 'gnb'){
    		 curDeleteGroupUrl = '${ctx}/gnb/pm/indicatormg/delIndicatorGroup.action';
    	 }else{
    		 curDeleteGroupUrl = '${ctx}/egw/pm/indicatormg/delIndicatorGroup.action';
    	 }
    	 
    	 setTimeout(function(){
	    	 var node=$('#kpiArithmeticTree').tree("getSelected");
	    	 $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingShanChuZhiBiaoGongNengJi")%>", function (r) {
	 	        if (r) {
	 	           $.post(curDeleteGroupUrl, {"catagoryId": node.id}, function(data) {
	 	    			if (data["success"]) {
	 	    				$("#kpiArithmeticTree").tree("reload");
	 	                } else {
	 	                	showMsg('error_msg',data["message"]);
	 	                    return;
	 	                }
	 	            }, "json");
	 	        }
	 	    }).addClass('seriousConfirm');
    	},300);
     }
     
     function addSign(sign){
    	$("#calcExpValueShowTitle").html("");
     	if(sign==='clear'){
     	    $('#calcExpValue').val('');
     	    $('#calcExpValueShow').val('');
     	    $('#calcExpValueShow').html('');
     	    $('#productAll').val('');
     	    itemArr = new Array();
     	    nameArr = new Array();
     	    productArr = new Array();
     	    updateCalcExpValue({});
     	}else if(sign==='num'){
     		var params = {key:'num',value: '',isNum:true,needCheck:true,name: ''};
            addRuleDom('#calcExpValueShow',params,updateCalcExpValue,true);
     	}else{
     		var oldVal=$('#calcExpValue').val();
             $('#calcExpValue').val(oldVal+sign);
             
             var oldShowVal=$('#calcExpValueShow').val();
             var params = {key:sign,value:sign,name:sign};
             addRuleDom('#calcExpValueShow',params,updateCalcExpValue);
     	}
     }

	function beforeLoad_kpiDatagrid(param){
		if(kpiManagePageVue.currentKpiNetType == 'enb'){
			 param["product_type"] = tableProductTypeSelect;
			 //level
		  	param.indicatorLevel = kpiManagePageVue.levelSelectParam;
			 if(writableMap['CODE_PERFORMANCE_MANAGEMENT']){
				$("#kpiDatagrid").datagrid("showColumn","ck");
				if(batchOperation){
					singleArr[0] = false;
				 }else {
					singleArr[0] = true;
				 }
				 $("#kpiDatagrid").datagrid('options').singleSelect = singleArr[0];

			 }else{
				$("#kpiDatagrid").datagrid("hideColumn","ck");
			 }
			 
			 
			 $("#kpiDatagrid").datagrid("showColumn","isEnable");
			 $("#kpiDatagrid").datagrid("showColumn","indicatorLevel");
   	    }else if(kpiManagePageVue.currentKpiNetType == 'gnb'){
   		     //5g 参数
   		     //param["product_type"] = 'BBU_XSS';
   		  	 $("#kpiDatagrid").datagrid("hideColumn","ck");
			 $("#kpiDatagrid").datagrid("hideColumn","isEnable");
			$("#kpiDatagrid").datagrid("hideColumn","indicatorLevel");
   	    }else{
   	   		//eGW 参数
  		     //param["product_type"] = 'WCG';
  		     $("#kpiDatagrid").datagrid("hideColumn","ck");
			 $("#kpiDatagrid").datagrid("hideColumn","isEnable");
			 $("#kpiDatagrid").datagrid("hideColumn","indicatorLevel");
   	    }
		param["searchText"] = kpiManasearchText;
		param["queryType"] = 'mgmt';
		param["timeZone"] = timeZone;
	}
	var itemArr = new Array(),nameArr = new Array(), productArr = new Array();
	function updateCalcExpValue(params){
	    if(params.type == 'add'){
	      	itemArr.splice(params.index+1,0,params.key);
	      	nameArr.splice(params.index+1,0,params.name);
	      	if(kpiManagePageVue.currentKpiNetType == 'enb'){
	    	 	 productArr.splice(params.index+1,0,params.product_type);
	      	}else{}
	      
	    }else if(params.type == 'remove'){
	      itemArr.splice(params.index,1);
	      nameArr.splice(params.index,1);
	     
	      //if(kpiManagePageVue.currentKpiNetType == 'gnb'){
	    	  productArr.splice(params.index,1);
	      //}else{}
	    }else if(params.type == 'update'){
	        itemArr.splice(params.index,1,params.key);
	        nameArr.splice(params.index,1,params.key);
	    }
	    $('#calcExpValue').val(itemArr.toString().replace(/[,]/g, ""));
	    //回显当前计算公式中指标对应的指标名称
	    var textstr = '';
	    nameArr.map(function(text,idx){
	    	textstr += '<span title="'+itemArr[idx]+'" class="kpi-text-node">'+text+'</span> ';
	    });
	    $('#kpiAlgorithmicOperNameDiv').html(textstr);
	    
	    //4G 需获取被选中指标的交集传给后端
		/*
			处理已选指标对应的产品类型说明:
			1、若某个指标的产品类型为 ALL，则表示该指标适用于所有产品类型，不影响交集结果;
			2、在计算交集时，忽略 ALL，只考虑具体的产品类型。
			--------------------------------------------------------------------------
			计算公式中所选指标的产品类型 无 ALL 的情况：
			1、均无产品类型：
				eg: 加减乘除，未选择任何指标； 实际下发后，接口也会提示计算公式错；
				product_type: 'ALL';
			2、产品类型有交集-交集+去重：
				eg: A：QRTB/RTS *  B:MLN/MLQ/BM/QAFB/CR-B4860/QRTB/QAFA/BAIBLQ/RTS
				product_type: 'QRTB/RTS'; 
			3、产品类型无交集-提示不一致：
				eg: A:MLN/MLQ/BM/QAFB/CR-B4860  * B:QRTB/RTS
				提示：已选指标的产品类型不一致；
			--------------------------------------------------------------------------
			计算公式中所选指标的产品类型 有 ALL 的情况：
			1、均为 ALL：
				eg: A:ALL + B:ALL
				product_type: 'ALL';
			2、ALL + 有交集的产品类型 = 交集+去重：？这种情况是否正确
				eg: A：QRTB/RTS *  B:MLN/MLQ/BM/QAFB/CR-B4860/QAFA/BAIBLQ/QRTB/RTS + C: ALL
				product_type: 'QRTB/RTS';
			3、ALL + 其它产品类型-无交集：
				3-1、eg: A:MLN/MLQ/BM/QAFB/CR-B4860  * B:QRTB/RTS + C:ALL
				提示：已选指标的产品类型不一致；
				3-2、eg: A:ALL + BSC;
				product_type: 'BSC';
		
		*/
	    if(kpiManagePageVue.currentKpiNetType == 'enb'){
		    if(productArr.length > 1){
		    	var hasALL = false; // 是否包含 ALL
		    	var normalProducts = []; // 非 ALL 的产品类型数组
		    	var allProductArr = []; // 所有产品类型的二维数组
		    	
		    	// 第一步：分类处理，区分 ALL 和具体产品类型
	    		productArr.map(function(item){
					if(typeof(item) != 'undefined' && item != ''){ 
						if(item == 'ALL'){
							hasALL = true;
						}else{
							var products = item.split(',');
							normalProducts.push(products);
							allProductArr.push(products);
						}
					}
				});
				
				// 第二步：根据不同情况处理
				if(normalProducts.length == 0){
					// 情况1：均无产品类型（只有加减乘除运算符，未选择任何指标）
					intersetionTip = false;
					$('#productAll').val('ALL');
				}else if(hasALL && normalProducts.length > 0){
					// 有 ALL 的情况：计算非 ALL 产品的交集
					var productIntersetion = function(arrs){
			    		return arrs.reduce(function(prev,cur){
			    			return Array.from(new Set(cur.filter((item)=>prev.includes(item))))
			    		})
			    	};
			    	var resultProduct = productIntersetion(normalProducts);
			    	
			    	if(resultProduct.length == 0){
			    		// 情况：ALL + 其它产品类型无交集
			    		intersetionTip = true;
						showMsg('error_msg','<%=rb.getString("YiXuanZhiBiaoDeChanPinLeiXingBuYiZhi")%>');
			    	}else{
			    		// 情况：ALL + 有交集的产品类型 = 交集
			    		intersetionTip = false;
			    		$('#productAll').val(resultProduct.toString().replace(/[,]/g, "/"));
			    	}
				}else{
					// 无 ALL 的情况：直接计算交集
					var productIntersetion = function(arrs){
			    		return arrs.reduce(function(prev,cur){
			    			return Array.from(new Set(cur.filter((item)=>prev.includes(item))))
			    		})
			    	};
			    	var resultProduct = productIntersetion(normalProducts);
			    	
			    	if(resultProduct.length == 0){
			    		// 产品类型无交集
			    		intersetionTip = true;
						showMsg('error_msg','<%=rb.getString("YiXuanZhiBiaoDeChanPinLeiXingBuYiZhi")%>');
			    	}else{
			    		// 产品类型有交集
			    		intersetionTip = false;
			    		$('#productAll').val(resultProduct.toString().replace(/[,]/g, "/"));
			    	}
				}
		    }else if(productArr.length == 1){
		    	// 只有一个指标的情况
		    	intersetionTip = false;
		    	var product = productArr[0];
		    	if(product == 'ALL' || typeof(product) == 'undefined' || product == ''){
		    		$('#productAll').val('ALL');
		    	}else{
		    		$('#productAll').val(product.replace(/[,]/g, "/"));
		    	}
		    }else if(productArr.length == 0){
		    	// 没有选择任何指标
		    	intersetionTip = false;
		    	$('#productAll').val('ALL');
		    }
	    }
	}
	function closekpiManaTemplate(){
		$("#kpiManaTemplate").animate({right:'-2000px'},500,function(){
			$(this).html("");
		});		
	}
	function kpiAddCancel(){
    	$("#kpiManaAdd").slideUp(300,function(){
    		$(this).html("");
    	});
    	try{
    		itemArr = new Array();
    		nameArr = new Array();
    		productArr = new Array();
    	}catch(e){}
	}
	
	function kpiModifyCancel(){
		$("#kpiManaAddOrModify").animate({right:'-1200px'},500,function(){
    		$(this).html("");
    	});
    	try{
    		itemArr = new Array();
    		nameArr = new Array();
    		productArr = new Array();
    	}catch(e){}
	}
	//导出所有指标
	function exportKpiInfo(){
	    var curExportUrl = '',curProduct = '';
	    var params = {
	    		timeZone: timeZone,
	    		searchText: kpiManasearchText
	    };
	    
	    if(kpiManagePageVue.currentKpiNetType == 'enb'){
	    	curExportUrl = '${ctx}/pm/indicatormg/exportAllIndicator.action';
	    	params.product_type = tableProductTypeSelect;
			//level
			params.indicatorLevel = kpiManagePageVue.levelSelectParam;
	    }else if(kpiManagePageVue.currentKpiNetType == 'gnb'){
	    	curExportUrl = '${ctx}/gnb/pm/indicatormg/exportAllIndicator.action';
	    	//curProduct = 'BBU_XSS';
	    }else{
	    	curExportUrl = '${ctx}/egw/pm/indicatormg/exportAllIndicator.action';
	    	//curProduct = 'WCG';
	    }
	    exportByForm(curExportUrl,params);
	}
/**
 * selector: 监听的容器
 * params: 添加的数据，格式{key:'xxx',value:'xxx',type:'xxx'}
 * fn: 回调函数
 * editable: 添加的节点是否可编辑（Boolen）
 **/
function addRuleDom(selector,params,fn,editable){
    var editable = editable || ((typeof fn == 'boolean')?fn:false);
    params.type = 'add';
    var ruleNode = domRule(params,editable);
    var nodePointer = domPointer(params);
    var container = document.querySelector(selector),
        focusNode = document.querySelector('.pointer.checked'),
        items = container.querySelectorAll('.pointer');
    params.index = items.length;
    if(focusNode){
      for(var i=0;i<items.length;i++){
        if(items[i] == focusNode) params.index = i;
      }
      var parentNode = focusNode.parentNode;
      parentNode.insertBefore(ruleNode,focusNode);
      parentNode.insertBefore(nodePointer,ruleNode);
      focusNode.focus();
    }else{
      container.appendChild(ruleNode);
      container.appendChild(nodePointer);
    }
    if(editable && params.isNum && params.needCheck) {
    	ruleNode.querySelector('.ivu-tag-text').click();
    	ruleNode.querySelector('.ivu-tag-text').focus();
    }
    if(typeof fn == 'function') try {fn(params);} catch (e) {};
	/* 创建规则dom节点 */
    function domRule(obj,bool){
      var cloneDom = document.querySelector('#hiddenTag').cloneNode(true),
          isEditable = bool || false;
      cloneDom.style.display = 'inline-block';
      cloneDom.firstElementChild.innerHTML = obj.value;
      cloneDom.firstElementChild.setAttribute('contenteditable',isEditable);
      cloneDom.addEventListener('click',function(event){
        if(event.target.tagName == 'I') {
          var divs = container.querySelectorAll('div');
          params.index = divs.length;
          for(var i=0;i<divs.length;i++){
            if(divs[i] == this) params.index = i;
          }
          event.target.parentNode.nextElementSibling.remove();
          event.target.parentNode.remove();
          obj.type = 'remove';
          if(typeof fn == 'function') try {fn(obj);} catch (e) {};
        }
      });
      if(obj.isNum && bool){
    	var spanDom = cloneDom.querySelector('span');
    	cloneDom.setAttribute('old',spanDom.innerText);
    	
    	function getSelectText(){
			var userSelection, text;
			userSelection = window.getSelection || document.selection.createRange();// not IE || IE
			if(!(text = userSelection.text)) text = userSelection();
			
			return text;
		}
    	cloneDom.addEventListener('keydown',function(event){
    		$(spanDom).tooltip({position:'right',content:'0.00001~9999999.99999'}).tooltip('destroy');
    		var keyCodes = ['8','37','39','48','49','50','51','52','53','54','55','56','57','91',
    		                '96','97','98','99','100','101','102','103','104','105','110','190'];
    		if($.inArray(event.keyCode+'',keyCodes)>=0){
    			var valueStr = this.innerText+'',
    				valueText = parseFloat(valueStr)+'',
    				dotIndex = valueStr.indexOf('.');
    			var selectText = getSelectText(),
    				baseIndex = selectText.baseOffset,
    				focusIndex = selectText.focusOffset,
    				maxIndex = Math.max(baseIndex,focusIndex);
    			var preventable = false;
    			if((event.keyCode == 110 || event.keyCode==190) && dotIndex>=0){
    				preventable = true;
    			}
    			if(dotIndex>=0 && !preventable){// 小数
    				var prevTxt = valueStr.split('.')[0],
    					preValue = parseFloat(prevTxt)+'',
    					afterTxt = valueStr.split('.')[1],
    					afterValue = parseFloat('.'+afterTxt)+'';
    				if(maxIndex<=dotIndex){// 光标在小数点前
    					var afterText = prevTxt.substring(maxIndex)+'';
        				if((preValue.length>=7 || afterText.length>=7) && !(event.keyCode == 110 || event.keyCode==190)){
        					preventable = true;
        				}
    				}else{// 光标在小数点后
    					var beforeTxt = valueStr.substring(dotIndex+1,maxIndex);
        				if((afterValue.length>=7 || beforeTxt.length>=5) && !(event.keyCode == 110 || event.keyCode==190)){
        					preventable = true;
        				}
    				}
    			}else{// 整数
    				var afterText = valueStr.substring(maxIndex)+'';
    				if((valueText.length>=7 || afterText.length>=7) && !(event.keyCode == 110 || event.keyCode==190)){
    					preventable = true;
    				}
    			}
    			if(preventable && event.keyCode!=8 && event.keyCode!=37 && event.keyCode!=39 ){
        			event.preventDefault();
        			event.stopPropagation();
        			$(spanDom).tooltip({position:'right',content:'0.00001~9999999.99999'}).tooltip('show');
    			}
    		}else{
    			event.preventDefault();
    			event.stopPropagation();
    			$(spanDom).tooltip({position:'right',content:'0.00001~9999999.99999'}).tooltip('show');
    		}
    	});
        cloneDom.addEventListener('input',function(event){
        	try{
                $(spanDom).tooltip({position:'right',content:'0.00001~9999999.99999'}).tooltip('destroy');
        	}catch(e){}
            var divs = container.querySelectorAll('div');
            params.index = divs.length;
            for(var i=0;i<divs.length;i++){
              if(divs[i] == this) params.index = i;
            }
            var textValue = this.innerText;
            if(!isNaN(textValue) && parseFloat(textValue) < 10000000 && parseFloat(textValue) >= 0){
            	obj.type = 'update';
                obj.key = parseFloat(this.innerText.trim());
                if(typeof fn == 'function') try {fn(obj);} catch (e) {};
                this.setAttribute('old',spanDom.innerText);
            }else if(textValue){
                spanDom.innerText = this.getAttribute('old');
        		$(spanDom).tooltip({position:'right',content:'0.00001~9999999.99999'}).tooltip('show');
            }
        });
        spanDom.addEventListener('focus',function(event){
        	numLayoutMask();
        	try{
                $(this).tooltip({position:'right',content:'0.00001~9999999.99999'}).tooltip('destroy');
        	}catch(e){}
        });
        spanDom.addEventListener('blur',function(event){
            if(this.innerText) {
            }else{
            	this.click();
            	this.focus();
            	numLayoutMask(true);
            }
            try{
                $(this).tooltip({position:'right',content:'0.00001~9999999.99999'}).tooltip('destroy');
        	}catch(e){}
        });
      }
      return cloneDom;
    }
	function numLayoutMask(bool){
		if(bool) {
			$("<div class='num-mask-layout' style='width:100%;height: 100%;background-color: rgba(255,255,255,0.3);position:absolute;z-index: 666;'></div>").appendTo($("body"));
			setTimeout(function(){
				numLayoutMask();
			},100);
		}else $('.num-mask-layout').remove();
	}
	/* 创建光标dom节点 */
	function domPointer(obj){
		var node = document.createElement('input');
		node.style.width = '2px';
		node.style.border = 'none';
		node.style.outline = 'medium';
		node.style.padding = '2px 3px 2px 5px';
		node.style.marginRight = '2px';
		node.className = 'pointer';
		node.addEventListener('keyup',function(event){
			if(event.keyCode =='8'){
				var inputs = container.querySelectorAll('.pointer');
				params.index = inputs.length;
				for(var i=0;i<inputs.length;i++){
					if(inputs[i] == this) params.index = i;
				}
				var prevAreaNode = node.previousElementSibling,
					prevPointer = prevAreaNode.previousElementSibling;
				if(prevPointer) {
					prevPointer.click();
					prevPointer.focus();
				}
				prevAreaNode.remove();
				node.remove();
				obj.type = 'remove';
				if(typeof fn == 'function') try {fn(obj);} catch (e) {};
			}else{
				node.value = '';
			}
		});
		node.addEventListener('click',function(event){
			for (var sibling of this.parentNode.querySelectorAll('input.pointer')) {
				if(sibling == this) this.className = 'pointer checked';
				else if(sibling.className == 'pointer checked') sibling.className = 'pointer';
			}
		});
		return node;
	}
}
function checkValidate(){
	var ele = $('.checkValidate');
	$.each(ele,function(index,item){
		$(item).validatebox('validate');
	})
}
// 反向显示
function formatRuleReview(selector,params,fn){
	function formateRuleStr(str){
		return str.replace(/([\+\-*\/\(\)])/g,'`$1`').replace(/(`+)/g,'`').replace(/^`/, '').replace(/`$/, '').split('`');
	}
    
	if(kpiManagePageVue.currentKpiNetType == 'enb'){
		var keys = params.keys,
			values = params.values,
			names = params.names,
			product_types = '',
			paramArr = new Array(),
			// 防御性检查：确保 oldKpiData 存在，如果不存在则使用空对象
			// oldKpiData 在 kpi_modifyKpi.jsp 中定义，但在查看详情时可能未加载
			oldKpiDatas = (typeof oldKpiData !== 'undefined' && oldKpiData && oldKpiData.indicatorProductRela) ? oldKpiData.indicatorProductRela : {};
		
		for (var i = 0; i < keys.length; i++) {
			product_types = oldKpiDatas[keys[i]] || '';
			
			// 统一将产品类型中的斜杠和分号转换为逗号，保持一致性
			if(product_types && typeof product_types === 'string') {
				product_types = product_types.replace(/\//g,',');
			}
			if(keys[i]) paramArr.push({
				key: keys[i],
				value: values[i],
				name: names?names[i]:'', 
				product_type: product_types
			});
		}
    }else {
    	var keys = params.keys,
			values = params.values,
			names = params.names,
			paramArr = new Array();
		for (var i = 0; i < keys.length; i++) {
			if(keys[i]) paramArr.push({key: keys[i],value: values[i],name: names?names[i]:''});
		}
    }
	
	for (var k = 0; k < paramArr.length; k++) {
		var editable = false;
		if(paramArr[k].key == paramArr[k].value && !isNaN(paramArr[k].key)) {
			editable = true;
			paramArr[k].isNum = true
		}
		addRuleDom(selector,paramArr[k],fn,editable);
	}
}
/*  选取当前颜色值  */
function choseColor(ele){	
	$('.color').hide();
	$(ele).next().show();   
    var $a = $(ele).next().children().children();
    $a.each(function(){
        $(this).click(function(){
            $('.color').hide();
            var cor = $(this).getHexBackgroundColor();
            $(ele).css('background',cor);
            var eleId=$(ele).attr("id");
            if(cor=="#ffffff"){
          	  $("#kpiThresholdForm input[name='"+eleId+"']").val("");
          	  $("#kpiThresholdForm input[name='"+eleId.replace("color","Begin")+"']").val("");
          	  $("#kpiThresholdForm input[name='"+eleId.replace("color","End")+"']").val("");
            }else{
          	  $("#kpiThresholdForm input[name='"+eleId+"']").val(cor);
            }
        })
    })
}
//关闭修改和新建指标功能集弹窗
function closeKpiGroupFun(){
	closeDefaultWindow();
}
</script>