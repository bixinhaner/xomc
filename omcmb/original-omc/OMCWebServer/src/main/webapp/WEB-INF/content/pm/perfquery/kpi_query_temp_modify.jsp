<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>

<style type="text/css">
	#modify_temp_device_datagrid #prefix_modify_temp_device_datagrid .datagrid-body { height: 306px !important; }
	.el-pairgrid-title{ top: -26px; }
	.productTypeSelect .el-select .el-input__inner{ height: 32px !important; line-height: 32px; }
	.el-form-item{ margin-bottom: 26px; }
	.querygroup { height: 32px; }
	.selectLimit { margin-left: 20px; color: #4D84FF; }
</style>
<div class="pageDefault addPage" id="updateTemplate">
<!-- 修改模板 -->
	<div class="slidebarTitleDiv">
		<span id="modifyTemplateTitle"><%=rb.getString("XiuGai")%></span>
		<a class="el-icon el-icon-close slideIcon" onclick="closeModifyTemplateDiv()" style="position: absolute; top:5px; right: 20px;font-size: 30px;"></a>
	</div>
	<div id="modifyTemplateDiv_body" class="slideBody">
	<div class="slideCont">
		<div class="splitGroup">
			<div class="splitGroup_title"><%=rb.getString("JiBenXinXi")%></div>
	  		<div class="splitGroup_body">			
				<div style="height:70px;">						
					<label class="inputTittleCss"><%=rb.getString("MuBanMingCheng")%></label>
					<input id="modifyTempName" maxlength="50" class="inputDivCss border border-box" style="width: 400px;"/>
					<div class="errorTitle" id="modifyTempName_err"><%=rb.getString("QingShuRuMuBanMingCheng")%></div>	
				</div>
				<div >					
					<label class="inputTittleCss"><%=rb.getString("MiaoShu")%></label>
					<textarea id="modifyTempDescript" maxlength="500" class="inputDivCss border border-box" style="width:800px;height:130px;vertical-align:top;resize:none;"></textarea>
				</div>		
			</div>    		
	   </div>
	   <div class="splitGroup">
			<div class="splitGroup_title"><%=rb.getString("ZhouQiSheDing")%></div>
			<div class="splitGroup_body" style="border: 1px solid #E2E3E9;width: 850px;display:flex;">
				<div style="width: 140px;border-right: 1px solid #E9E9E9;background:#FDFDFD;">
					<div class="periodHeader" style="text-align: center;"><%=rb.getString("ChaXunLiDu")%></div>
					<div id="modifyTempPeriod" class="flex-ctn">
						<span class="period-item">
							<input type="radio" name="period" value="15"/><label> 15Min</label>
						</span>
						<span class="period-item">
							<input type="radio" name="period" value="60"/><label> 60Min</label>
						</span>
						<span class="period-item">
							<input type="radio" name="period" value="1440"/><label> 24hour</label>
						</span>
					</div>
					<label class="inputTipCss errorTipStyle"></label>
				</div>
	   			<div style="flex-grow:1;">
					<div id="modifyKpiTimeSlotDiv" class="periodHeader">
						<label style="margin:0 50px 0 20px;"><%=rb.getString("ChaXunShiDuan")%></label>
						<input type="radio" name="timeSlot" id="allTimeSlot" checked value="1"><label for="allTimeSlot"><%=rb.getString("SuoYouShiDuan")%></label>
						<input type="radio" name="timeSlot" id="partTimeSlot" value="2" class="not_support"><label class="not_support" for="partTimeSlot"><%=rb.getString("ZhiDingShiDuan")%></label>
					</div>
					
					<div id="modifyTimeSlotSelectDiv" style="padding:20px;" class="period_readonly">
						<div>
							<span class="timeSlotTitle"><%=rb.getString("Zhou")%><%=rb.getString("MaoHao")%></span>
							<div class="selectRegion_week">
								<p value="0">Sun</p>
								<p value="1">Mon</p>
								<p value="2">Tues</p>
								<p value="3">Wed</p>
								<p value="4">Thurs</p>
								<p value="5">Fri</p>
								<p value="6">Sat</p>
							</div>
						</div>
						<div style="margin-top:20px;">
							<span class="timeSlotTitle"><%=rb.getString("XiaoShi")%><%=rb.getString("MaoHao")%></span>
							<div class="selectRegion_hour">
								<p value="0">0</p>
								<p value="1">1</p>
								<p value="2">2</p>
								<p value="3">3</p>
								<p value="4">4</p>
								<p value="5">5</p>
								<p value="6">6</p>
								<p value="7">7</p>
								<p value="8">8</p>
								<p value="9">9</p>
								<p value="10">10</p>
								<p value="11">11</p>
								<p value="12">12</p>
								<p value="13">13</p>
								<p value="14">14</p>
								<p value="15">15</p>
								<p value="16">16</p>
								<p value="17">17</p>
								<p value="18">18</p>
								<p value="19">19</p>
								<p value="20">20</p>
								<p value="21">21</p>
								<p value="22">22</p>
								<p value="23">23</p>
							</div>
						</div>
					</div>
					<label class="inputTipCss errorTipStyle"></label>
				</div>
			</div>
	   </div>
	   <div class="splitGroup" id="group_device_div">
			<div class="splitGroup_title"><%=rb.getString("SheBeiLieBiao")%></div>
			<div style="display: flex;flex-direction: column;">
				<div style="margin-left: 54px;margin-top: 20px;">
					<span><%=rb.getString("SheBeiXuanZheFangShi")%></span>
				</div>
				<div id="modifyTempDeviceType" style="display: flex;flex-direction:column;width: 90%;border: 1px solid #E9E9E9;margin-left: 54px;margin-top:5px" >
					<el-form ref="form" :model="form" label-position="top" style="flex: auto; overflow: auto;">
			   			<div class="selectListCont" style="padding: 10px 20px 0;">
							<el-form-item prop="selectType" class="deviceSelectType">
								<div>
									<el-radio-group v-model="form.selectType">
										<el-radio label="1"><%=rb.getString("SheBeiZu")%></el-radio>
										<el-radio label="2"><%=rb.getString("KPISheBei")%></el-radio>
									</el-radio-group>
									<span class="selectLimit" style="margin-left: 20px;"><%=rb.getString("ZuiDuoXuanZe")%>  {{deviceNumLimit}} <%=rb.getString("ZuiDuoXuanZeDevice")%></span>
								</div>
							</el-form-item>
							<!-- 设备组列表 -->
							<el-form-item v-show='showDeviceGroup' prop="groups"> 
								<el-ctable :id="'selected_device_list'" :limit="20"
								ref="groupTable" 
								:row-key="'id'" 
								:default-checked="defaultCheckedGroup" 
								:url="groupUrl" 
								:height="'360px'" 
								:pagination="false" 
								:rownumber="true" 
								@selection-change="groupChange" 
								style="border:1px solid #E9E9E9;">
									<el-table-column label='' type="selection" :reserve-selection="true"></el-table-column>
									<el-table-column label='<%=rb.getString("SheBeiZuMingCheng")%>' prop="group_name"></el-table-column>
								</el-ctable>
								<p v-show="deviceGroupShow" style="color: #FA5555; font-size: 12px;height: 12px;"><%=rb.getString("QingXuanZeSheBeiZu")%></p>
							</el-form-item>
							<!-- 设备列表 -->
							<el-form-item v-show='!showDeviceGroup'> 
								<el-pairgrid :id="'select_device_list'" 
								:rownumber="true" 
								ref="cpairgrid" 								
								:right-url="rightUrl" 
								:left-url="leftUrl" 
								:height="'360px'" 
								:row-key="'serialNumber'" 
								:query-params="queryForm" 
								:title="deviceTitle"
								:limit="deviceNumLimit"
								:messages="commonMessage" 
								@checked-change="devicesChange">
									<template slot="prev">
										<el-ctable style="width:300px;" :id="'group_list'" :show-pager="false" ref="group" :url="groupUrl" :height="'100%'" :show-header="false" :row-key="'id'" @current-change="queryGroupChange" :pagination="false">
											<template slot='toolbar'>
												<%=rb.getString("SheBeiZu")%>
											</template>
											<el-table-column label='<%=rb.getString("SheBeiZu")%>' prop="group_name"></el-table-column>
										</el-ctable>
									</template>
									<template slot="left">
										<el-table-column type="selection" width="45"></el-table-column>
										<el-table-column v-if="selectModelType == '0' || selectModelType == '1'" prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
										<el-table-column v-if="selectModelType == '0' || selectModelType == '1'" prop='hostName' label='<%=rb.getString("HostName")%>'></el-table-column>
										<el-table-column v-if="selectModelType == '2'" prop='serialNumber' label='<%=rb.getString("eGWBianMa")%>'></el-table-column>
										<el-table-column v-if="selectModelType == '2'" prop='hostName' label='<%=rb.getString("eGWMingCheng")%>'></el-table-column>
										<el-table-column v-if="selectModelType == '0'" prop='product' label='<%=rb.getString("ChanPinLeiXing")%>'></el-table-column>
									</template>
									<template slot='toolbar'>
										<div style="display: flex;">
											<el-form-item label="" v-if="selectModelType == '0'" style="display:flex; margin-left: 20px;" class="productTypeSelect">
												<el-select v-model="product_type" @change='productChange'>
													<el-option v-for="item in productTypeList" :key="item.value" :label="item.label" :value="item.value"></el-option>
												</el-select>
											</el-form-item>
											<div class="queryGroup">
												<el-input class='pairgrid-query' v-model="search_text" @keyup.enter.native="deviceQuery"
													:placeholder="placeholderSelect"></el-input>
										    	<i @click='deviceQuery' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
											</div>
										</div>
									</template>
									<template slot='right'>
										<el-table-column v-if="selectModelType == '0' || selectModelType == '1'" prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
										<el-table-column v-if="selectModelType == '0' || selectModelType == '1'" prop='hostName' label='<%=rb.getString("HostName")%>'></el-table-column>
										<el-table-column v-if="selectModelType == '2'" prop='serialNumber' label='<%=rb.getString("eGWBianMa")%>'></el-table-column>
										<el-table-column v-if="selectModelType == '2'" prop='hostName' label='<%=rb.getString("eGWMingCheng")%>'></el-table-column>										
									</template>
								</el-pairgrid>
								<p v-show="deviceShow" style="color: #FA5555; font-size: 12px; height: 12px;"><%=rb.getString("QingXuanZeSheBei")%></p>
							</el-form-item>
							<el-form-item prop='devices' style="margin-left:45px;" label-width="0">
								<el-input v-model='form.devices' v-show="false"></el-input>
							</el-form-item>
						</div>
			   		</el-form>
					
		   		</div>
	   		</div>
	   	</div>
	  	<div class="splitGroup">
	  		<div class="splitGroup_title">
	  			<%=rb.getString("ZhiBiaoXuanZe")%>
	  			<span style="margin-left:10px;color:#4D84FF;font-size:12px;font-weight:normal;"><%=rb.getString("ZuiDuoXuanZe")%> <span class="kpiLimit"></span> <%=rb.getString("ZuiDuoXuanZeKPI")%></span>
	  		</div>
	  		<div style="width:90%;height:350px;margin-left:54px;">
	  			<div id="modify_temp_kpi_datagrid" ></div>
			</div>
			<label class="inputTipCss errorTipStyle" style="margin-left:54px"></label>    
			<label class="kpiTableLimitTip errorTipStyle" style="margin-left:54px;display:none;margin-top:-26px;"><%=rb.getString("ZuiDuoXuanZe")%> <span class=kpiLimit></span> <%=rb.getString("ZuiDuoXuanZeKPI")%></label>  				
	   	</div>
	   	</div>
	</div>
	<div class="slideFooter">
        <span class="el-button el-button--primary modifyTemp" onclick="confirmAddOrModifyTemplate()"><%=rb.getString("QueDing")%></span>
        <span class="el-button" onclick="closeModifyTemplateDiv()"><%=rb.getString("QuXiao")%></span>
        <input type="checkbox" name="is_default" value="true" id="temp_default_checkbox"/><label for="temp_default_checkbox"> <%=rb.getString("SheWeiMoRen")%></label>
   	</div>
</div>	

	<!-- 指标选择toolbar -->
	<div id="toolbar_modify_temp_kpi_datagrid" style="padding:5px 10px;">
		<input name="deviceGroup" id="tempModifyKpiGroup" class="easyui-combobox border border-box combobox-f combo-f textbox-f pairgridGroupFilter">
		<div class="queryGroup" style="margin:0 0 0 20px;">
			<input name="search_text" id="tempModifyKpiQuery" style="width:180px;" placeholder="<%=rb.getString("ZhiBiaoID")%> / <%=rb.getString("ZhiBiaoMingCheng")%>">
			<b class="el-icon el-icon-common-search" onclick="javascript: $('#modify_temp_kpi_datagrid').pairgrid('reload')"></b>
		</div>
	</div>

<script type="text/javascript">
	var oldTempData = {};
	var datagrid = $("#kpiTemplateDatagrid").datagrid("getSelected");  
	//初始化时选中的设备组
	var autoSelDeviceGroup = [];
	var groupSelectedList = [];
	var isAllOperator = false;
	var deviceLimitNum ='',
		kpiLimitNum = '';
	var curModifyEnbGnbEgwType = sessionStorage.getItem('modifyTemplateEnbGnbOrEgw');
	
	var addTemplate = new Vue({
		el:"#updateTemplate",
		data() {
		    return {
		    	queryForm: {
		    		tempId: '',
					groupId: '',
					searchText: '',
					product_type: ''
				},
				product_type: '',
				search_text:'',
				// 表单数据
				form: { 
					selectType: '2',
					groups: '',
					devices: ''					
				},
				deviceNumLimit: '',
				productTypeList: [],
				kpiList: [],
				maxRules: parseInt('${kpiAlarmThresholdMaxnum}'),
		    	leftUrl : '',
		    	rightUrl : '',//右侧已选数据
				groupUrl: '${ctx}/pm/template/getTempDeviceGroupList.action?tempId='+datagrid.tempId,//设备组
				deviceTitle: ['','<%=rb.getString("YiXuan")%>'],
				defaultCheckedGroup: [],
				curProductType:'',
				deviceGroupShow: false,
				deviceShow: false,
				deviceGroupSelection: [],
				selectModelType: '',
				placeholderSelect: "<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>",
				commonMessage: {
                    placeholder: '<%=rb.getString("XiaoZhanBianMa") %>'
                },
                tableCurProduct: '',
		    }
		 },
		 computed: {				
			showDeviceGroup(){ 
				return this.form.selectType == '1';
			}
		},
		watch: {
			"form.selectType":function(newVal){
				var vm = this;
				if(vm.selectModelType == '0'){
					if(newVal == '1'){
						vm.curProductType = vm.tableCurProduct;
					}else{
						var rows = vm.$refs.cpairgrid.getData();
						vm.commonProductType(rows);
					}
				}
			},
			curProductType: function(newVal,oldVal){
				var vm = this;
				if(vm.selectModelType == '0'){
					if(newVal !== oldVal){
						$("#modify_temp_kpi_datagrid").pairgrid("reload");
					}
				}
			}
		},
	    methods:{
	    	//0-4G, 1-5G, 2-eGW
			init(){				
				var vm = this, curSelDeviceGroupListUrl = '' ,params = {tempId: datagrid.tempId};
				
				vm.selectModelType = curModifyEnbGnbEgwType;
				
				vm.$nextTick(function(){
					vm.queryForm.tempId = datagrid.tempId;
					if(curModifyEnbGnbEgwType == '0'){
						vm.placeholderSelect = '<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>';
						vm.commonMessage =  {
			                placeholder: '<%=rb.getString("XiaoZhanBianMa") %>'
		                };
						vm.leftUrl = '${ctx}/pm/template/getModifyEnbListPageData.action';
						vm.rightUrl = '${ctx}/pm/template/getSelectedEnbListPageData.action?tempId='+datagrid.tempId;
					}else if(curModifyEnbGnbEgwType == '1'){  
						vm.placeholderSelect = '<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>';
						vm.commonMessage =  {
			                placeholder: '<%=rb.getString("XiaoZhanBianMa") %>'
		                };
						vm.leftUrl = '${ctx}/gnb/pm/template/getModifyEnbListPageData.action';
						vm.rightUrl = '${ctx}/gnb/pm/template/getSelectedEnbListPageData.action?tempId='+datagrid.tempId;
					}else{
						vm.placeholderSelect = '<%=rb.getString("eGWBianMa")%> / <%=rb.getString("eGWMingCheng")%>';
						vm.commonMessage =  {
			                placeholder: '<%=rb.getString("eGWBianMa") %>'
		                };
						vm.leftUrl = '${ctx}/egw/pm/template/getModifyEgwListPageData.action';
						vm.rightUrl = '${ctx}/egw/pm/template/getSelectedEgwListPageData.action?tempId='+datagrid.tempId;
					}
				});
				
				//设备组已选数据
				if(curModifyEnbGnbEgwType == '0'){
					curSelDeviceGroupListUrl = '${ctx}/pm/template/getSelDeviceGroupList.action';
					vm.getProductList();
				}else if(curModifyEnbGnbEgwType == '1'){  
					curSelDeviceGroupListUrl = '${ctx}/gnb/pm/template/getSelDeviceGroupList.action';
				}else{
					curSelDeviceGroupListUrl = '${ctx}/egw/pm/template/getSelDeviceGroupList.action';
				}
				
				axios.post(curSelDeviceGroupListUrl,stringify(params)).then(function(response){
					var data = response.data;
					if(data){
						var curArr = data.rows.map((item)=>{
							return item.id
						});
						autoSelDeviceGroup = curArr;
						autoSelDeviceGroup.sort();
						autoSelDeviceGroup = autoSelDeviceGroup.join(",");
						vm.defaultCheckedGroup = curArr;
						initForm(vm.$refs.form);
					}
				}).catch(function(error){});				
			},
			
			//获取已选设备的数据对应的产品类型
			getSelectRowsProduct(){
				var vm = this;
				//获取已选设备对应的产品类型，来动态更新指标数据
				axios.post('${ctx}/pm/template/getSelectedEnbListPageData.action?tempId='+datagrid.tempId).then(function(response){
					var data = response.data;
					if(data.rows){
						vm.commonProductType(data.rows);
					}				
				}).catch(function(error){})	
			},
			
			commonProductType(rows){
				var vm = this, curProductArr = [], newProductArr = [];
				
				(rows || []).map((item)=>{
					if(item.hasOwnProperty('product') && item.product != null &&　item.product != undefined && item.product != ''){
						curProductArr.push(item.product)	
					}						
				}); 
				if(curProductArr.length > 0){
					//产品类型去重
					curProductArr.map(function(item){
						if(newProductArr.indexOf(item) == -1){
							newProductArr.push(item)
						}
					});
					vm.curProductType = newProductArr.join(',');
				}else{
					vm.curProductType = vm.tableCurProduct;
					//kpi 列表调用了两次接口
				}
			},
			//获取产品类型
			getProductList(){
				var vm = this;
				//获取产品类型
				axios.post('${ctx}/cell/cpeinfos/getEnbMonitorProductList.action?isGnb=0&isAll=1').then(function(response){
					var data = response.data;
					if(data.length == 0){
						vm.queryForm.product_type = 'no';
					}else{
						//全部的产品类型
						vm.tableCurProduct = data.join(',');
						//搜索下拉框数据映射
						var arr = [{label:'<%=rb.getString("QuanBu")%>',value:''}];
						data.map(function(item){
							if (item){								
								arr.push({label:item,value:item})
							}
						});
						vm.productTypeList = arr;
					}
				}).catch(function(error){})
			},
			productChange(val){
				var vm = this;
				vm.queryForm.product_type = val;			
			},
			//devices 表格搜索
			deviceQuery(){
				this.queryForm.searchText = this.search_text;
			},
			queryGroupChange(row){ 
				if(row) {
					this.queryForm.groupId = row.id;
				}
			},
			// 设备组列表-当选择项发生变化时触发此事件
			groupChange(selection) {
				var vm = this;
				vm.$nextTick(function(){
					vm.deviceGroupSelection = selection;
					setTimeout(function(){
						var cks = vm.$refs.groupTable.getChecked();
						
						if(cks.length > 0){
							vm.deviceGroupShow = false;
						}else{
							vm.deviceGroupShow = true;
						}
					},0)
				})
			},
			// 设备选择变化时，更新选择设备记录
			devicesChange() {
				var vm = this;
				vm.$nextTick(function(){
					var rows = vm.$refs.cpairgrid.getData();
					if(rows){
						vm.form.devices = rows.map(function(row){ return row.smallCellCode ;}).join(',');
						if(vm.form.devices == ''){
							vm.deviceShow = true;
						}else{
							vm.deviceShow = false;
						}
					}	
					if(vm.selectModelType == '0'){
						vm.commonProductType(rows);
					}
				})		
			},
		 },
		 mounted(){
			this.init();			
		}		
	});
	
	$(function(){
		$.post("${ctx}/cell/perfmgmt/kpitemp/getLimitInfo.action",{}, function(data) {
			if(data){
				kpiLimitNum = parseInt(data.indicatorNum);
				$('.kpiLimit').html(kpiLimitNum);
				addTemplate.deviceNumLimit = parseInt(data.deviceNum);
			}		 							
		}, "json");
		
		//内置的Basic模板不允许修改名称和描述 
		if(datagrid.isCustomize == '0'){
			$('#modifyTempName').attr('disabled',true);
			$('#modifyTempDescript').attr('disabled',true);
		}
		
	    var params = {};
		params.tempId = datagrid.tempId;
		params.timeZone = timeZone;
		//模板信息
	    getKPITemlateInfo(params);
	    closeLoading();
	    
	  	//时间周期点击选择效果 
	    $(".selectRegion_week p , .selectRegion_hour p").each(function(){
	    	$(this).click(function(){
	    		$(this).toggleClass("active")
	    	})
	    })
	    
		//周期粒度切换  （当选择24小时粒度时，隐藏定制时段相关显示 ）
	    $("input[name='period']").on('click',function(){
	    	var allType = $("#modifyKpiTimeSlotDiv input[type=radio]:eq(0)").prop("checked");
	    	
	    	if(this.value == "1440"){
	    		if ( !allType){
	    			$("#modifyKpiTimeSlotDiv input[type=radio]:eq(0)").prop("checked",true);
	    			$('#modifyTimeSlotSelectDiv').addClass('period_readonly');
	    		}
	    		
				$(".not_support").css("visibility","hidden");
				$('#modifyTimeSlotSelectDiv').css("visibility","hidden");
		    	$("#modifyKpiTimeSlotDiv").next().next().html("");
			}else{
				$(".not_support").css("visibility","visible");
				$('#modifyTimeSlotSelectDiv').css("visibility","visible");
				
			}
	    });
	    
		//切换所有时段或是定制时段 （ 当前选择所有时段时，定制时间置灰不可点击 ）
	    $("#modifyKpiTimeSlotDiv input[type=radio]").on('click',function(){
	    	var sdiv = $('#modifyTimeSlotSelectDiv');
	    	
	    	if($("#modifyKpiTimeSlotDiv input:radio:checked").val() == 1){
		    		$('#modifyTimeSlotSelectDiv').addClass('period_readonly');
	    	}else {
	    		$('#modifyTimeSlotSelectDiv').removeClass('period_readonly');
	    	}
	    })
	    
	    var curModifyGroupListUrl = '';
	    if(curModifyEnbGnbEgwType == '0'){
	    	curModifyGroupListUrl = '${ctx}/pm/indicatormg/getIndicatorGroupList.action';	    	
	    }else if(curModifyEnbGnbEgwType == '1'){
	    	curModifyGroupListUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorGroupList.action';	    	
	    }else{
	    	curModifyGroupListUrl = '${ctx}/egw/pm/indicatormg/getIndicatorGroupList.action';
	    }
	    
	    //指标功能集下拉列表赋值 
	    $("#tempModifyKpiGroup").combobox({
	    	url: curModifyGroupListUrl,
	    	valueField: 'catagoryId',
			textField: 'catagoryName',
			queryParams: {
				isShowAll : "1",
				noKpiShowGroup : "0"
			},
			editable: false,
			onSelect: function(){
				$("#modify_temp_kpi_datagrid").pairgrid("reload")
			}
	    });
	  	
		//KPI选择列表 
		if(curModifyEnbGnbEgwType == '0'){
			setTimeout(function(){
				$('#modify_temp_kpi_datagrid').pairgrid({
			        idField: 'kpiId',
			    	leftUrl : '${ctx}/pm/indicatormg/getIndicatorListByPage.action',
			    	rightUrl : '${ctx}/pm/template/getSelectedIndicatorListPageData.action',
			        border:false,
			        fit:true,
			        fitColumns:true,
			        rownumbers : true,
			        striped : true,
			        singleSelect : true,
					pageList : [ 50, 100, 150, 200, 250, 300 ],
			        pagination: true,
			        pagePosition: 'bottom',
			        toolBar:"#toolbar_modify_temp_kpi_datagrid",
					zone : [50,50],
					queryName : 'kpiName,kpiId,catagoryName',
					messages:{queryName:'<%=rb.getString("ZhiBiaoID")%> / <%=rb.getString("ZhiBiaoMingCheng")%> / <%=rb.getString("ZhiBiaoGongNengJi")%>'},
					onCheck: modifyKpiDatagridCheck,
			        leftBeforeLoad : beforeLoad_modify_temp_left_kpi_datagrid,
			        rightBeforeLoad : beforeLoad_modify_temp_right_kpi_datagrid,
			        onLoadSuccess : datagridLoadSuccess,
			  	    leftColumns : [
			  	    	{field : 'ck',checkbox:true},
						{field:"kpiId",sortable : true,width : 100,title: '<%=rb.getString("ZhiBiaoID")%>'},
					    {field: "kpiName",sortable : true,title: '<%=rb.getString("ZhiBiaoMingCheng")%>',width:180},
					    {field: 'product_type',sortable: false,width: 120,title: '<%=rb.getString("ChanPinLeiXing")%>'},
						{field: 'catagoryName',sortable: true,hidden : true}
					],
					rightColumns : [
						{field : 'kpiId',title : '<%=rb.getString("ZhiBiaoID")%>(<%=rb.getString("ZhiBiaoMingCheng")%>)',width : 100,formatter: selectedKpiTableFun}, 
						{field: "kpiName",hidden : true},
						{field:"catagoryName",width :50,title: '<%=rb.getString("ZhiBiaoGongNengJi")%>'}
					]
			  	});
			},500);
		}else if(curModifyEnbGnbEgwType == '1'){
			$('#modify_temp_kpi_datagrid').pairgrid({
		        idField: 'kpiId',
		    	leftUrl : '${ctx}/gnb/pm/indicatormg/getIndicatorListPageData.action',
		    	rightUrl : '${ctx}/gnb/pm/template/getSelectedIndicatorListPageData.action',
		        border:false,
		        fit:true,
		        fitColumns:true,
		        rownumbers : true,
		        striped : true,
		        singleSelect : true,
				pageList : [ 50, 100, 150, 200, 250, 300 ],
		        pagination: true,
		        pagePosition: 'bottom',
		        toolBar:"#toolbar_modify_temp_kpi_datagrid",
				zone : [50,50],
				queryName : 'kpiName,kpiId,catagoryName',
				messages:{queryName:'<%=rb.getString("ZhiBiaoID")%> / <%=rb.getString("ZhiBiaoMingCheng")%> / <%=rb.getString("ZhiBiaoGongNengJi")%>'},
				onCheck: modifyKpiDatagridCheck,
		        leftBeforeLoad : beforeLoad_modify_temp_left_kpi_datagrid,
		        rightBeforeLoad : beforeLoad_modify_temp_right_kpi_datagrid,
		        onLoadSuccess : datagridLoadSuccess,
		  	    leftColumns : [
		  	    	{field: 'ck',checkbox:true},
		  	    	{field: "kpiId",sortable : true,width : 100,title: '<%=rb.getString("ZhiBiaoID")%>'},
		  	    	{field: "kpiName",sortable : true,title: '<%=rb.getString("ZhiBiaoMingCheng")%>',width:180},
		  	    	{field: 'catagoryName',sortable: true,hidden : true}
		  	    ],
				rightColumns : [
					{field : 'kpiId',title : '<%=rb.getString("ZhiBiaoID")%>(<%=rb.getString("ZhiBiaoMingCheng")%>)',width : 100,formatter: selectedKpiTableFun}, 
					{field: "kpiName",hidden : true},
					{field:"catagoryName",width :50,title: '<%=rb.getString("ZhiBiaoGongNengJi")%>'}
				]
		  	});
		}else{
			$('#modify_temp_kpi_datagrid').pairgrid({
		        idField: 'kpiId',
		    	leftUrl : '${ctx}/egw/pm/indicatormg/getIndicatorListPageData.action',
		    	rightUrl : '${ctx}/egw/pm/template/getSelectedIndicatorListPageData.action',
		        border:false,
		        fit:true,
		        fitColumns:true,
		        rownumbers : true,
		        striped : true,
		        singleSelect : true,
				pageList : [ 50, 100, 150, 200, 250, 300 ],
		        pagination: true,
		        pagePosition: 'bottom',
		        toolBar:"#toolbar_modify_temp_kpi_datagrid",
				zone : [50,50],
				queryName : 'kpiName,kpiId,catagoryName',
				messages:{queryName:'<%=rb.getString("ZhiBiaoID")%> / <%=rb.getString("ZhiBiaoMingCheng")%> / <%=rb.getString("ZhiBiaoGongNengJi")%>'},
				onCheck: modifyKpiDatagridCheck,
		        leftBeforeLoad : beforeLoad_modify_temp_left_kpi_datagrid,
		        rightBeforeLoad : beforeLoad_modify_temp_right_kpi_datagrid,
		        onLoadSuccess : datagridLoadSuccess,
		  	    leftColumns : [
		  	    	{field : 'ck',checkbox:true},
					{field:"kpiId",sortable : true,width : 100,title: '<%=rb.getString("ZhiBiaoID")%>'},
				    {field: "kpiName",sortable : true,title: '<%=rb.getString("ZhiBiaoMingCheng")%>',width:180},
					{field: 'catagoryName',sortable: true,hidden : true}
				],
				rightColumns : [
					{field : 'kpiId',title : '<%=rb.getString("ZhiBiaoID")%>(<%=rb.getString("ZhiBiaoMingCheng")%>)',width : 100,formatter: selectedKpiTableFun}, 
					{field: "kpiName",hidden : true},
					{field:"catagoryName",width :50,title: '<%=rb.getString("ZhiBiaoGongNengJi")%>'}
				]
		  	});
		}	    
	  	
	    //模板名称为空，给出提示
		$("#modifyTempName").blur(function(){
			if($(this).val().trim().length > 0){
				$("#modifyTempName_err").hide();
			}else{
				$("#modifyTempName_err").show();
			}
		});
		
		//指标选择 - 搜索回车事件
		$("#tempModifyKpiQuery").bind("keyup", function(e){
	        if (e.keyCode == 13){
	        	$("#modify_temp_kpi_datagrid").pairgrid("reload");
	        }
	    });
	})
		
	//指标选择，选中一行，提示内容清空
	function modifyKpiDatagridCheck(){
		$("#modify_temp_kpi_datagrid").parent().next().html("");
	}
	
	//指标选择-发送加载数据的请求前触发
	function beforeLoad_modify_temp_left_kpi_datagrid(param){
		
		if(curModifyEnbGnbEgwType == '0'){
			param.product_type = addTemplate.curProductType;
		}else if(curModifyEnbGnbEgwType == '1'){  
			//param.product_type = 'BBU_XSS';
		}else{
			//param.product_type = 'WCG';
		}
		param.catagoryId = $("#tempModifyKpiGroup").combobox("getValue");
		param.queryType = 'mgmt';
		param.searchText = $("#tempModifyKpiQuery").val();
	}
	
	//指标选择-已经选择-发送加载数据的请求前触发
	function beforeLoad_modify_temp_right_kpi_datagrid(param){
		param.tempId = datagrid.tempId;
	}

	/** 
	* 模板信息渲染
	* @param params[string] 选中行的模板 id
	**/
	function getKPITemlateInfo(params){
		var params = params, curModifyTemplateInfoUrl = '';
		
	    if(curModifyEnbGnbEgwType == '0'){
	    	curModifyTemplateInfoUrl = '${ctx}/pm/template/getTemplateInfo.action';	    	
	    }else if(curModifyEnbGnbEgwType == '1'){
	    	curModifyTemplateInfoUrl = '${ctx}/gnb/pm/template/getTemplateInfo.action';
	    }else{
	    	curModifyTemplateInfoUrl = '${ctx}/egw/pm/template/getTemplateInfo.action';	
	    }
	    
		$.post(curModifyTemplateInfoUrl, params, function(data) {
			 if(data){
				 oldTempData = data;
				 $("#modifyTempName").val(data.tempName);			 
				 $("#modifyTempDescript").val(data.description);
				 //查询粒度点击
				 $("#modifyTempPeriod").find('input[value='+data.reportPeriod+']').click();
				 
				 if(data.isAllOperator == 1) {
					 isAllOperator = true;
					 $('#group_device_div').hide();
					 //此模板不显示设备，需将已勾选设备的产品类型置空
					 addTemplate.curProductType = '';
				 }
				 
				 if(data.is_default == "true") {
					 //$("#temp_default_checkbox").prop('checked','true');
					$('input[type="checkbox"]').prop('checked',true); //ok
				 }				 
				 
				 if(data.reportPeriod == "1440"){
					 
				 }else{
					 var week = data.week.split(",");
					 var hour = data.hour.split(",");
					 if(week[0] != "" && hour[0] != ""){
						 $("#modifyKpiTimeSlotDiv input[value='2']").click();
						 $('#modifyTimeSlotSelectDiv').removeClass('period_readonly');
						 //选中天
						 $.each($("#modifyTimeSlotSelectDiv .selectRegion_week p"),function(index,ele){
							 if($.inArray($(this).attr("value"),week) > -1) {
								 $(this).addClass("active");
		 	                 }
						 })
						 //选中小时点 
						 $.each($("#modifyTimeSlotSelectDiv .selectRegion_hour p"),function(index,ele){
							 if($.inArray($(this).html(),hour) > -1) {
								 $(this).addClass("active");
		 	                 }
						 })
						 
					 }else{
						 $("#modifyKpiTimeSlotDiv input[value='1']").click();
					 }
				 }
				 
				 //设备选择方式
				 addTemplate.form.selectType = data.selDeviceType;
				if(addTemplate.form.selectType == '1'){
					//设备组
					addTemplate.curProductType = addTemplate.tableCurProduct;
					addTemplate.defaultCheckedGroup = data.auto_access_device_group ? data.auto_access_device_group.split(',') : [];
				}else{
					//设备
					addTemplate.getSelectRowsProduct();
				}
				 // 往设备选择的右表静态载入数据并初始化级联
				 if(data.relDevice === '' || data.relDevice === null || data.relDevice === undefined){
				 }else{
					 addTemplate.form.devices = data.relDevice;	
				 }
				 
			 }else {
				 showMsg('error_msg',data["message"])
			 }
		 },"json");
	}        
	
	// 修改模板 确定事件
	function confirmAddOrModifyTemplate(){
		//标题的位置
		var firstOffset = $("#modifyTemplateTitle").offset().top;
		//模板
		var tempId = $("#kpiTemplateDatagrid").datagrid("getSelected").tempId;
		//是否为空，默认为true
		var isBlank = true;
		//参数
		var old_temp_name = $("#oldTemplateName").val();
		var old_description = $("#oldTemplateDescription").val();
		//基本信息
		var tempName = $("#modifyTempName").val().trim();
		var description = $("#modifyTempDescript").val().trim();
		//时间范围
		var reportPeriod = $("#modifyTempPeriod input:checked").val();
		var week = [];
		var hour = [];
		var checkedRadio = $("#modifyKpiTimeSlotDiv input[type=radio]:checked").val();
		var isDefault = $('#temp_default_checkbox').prop('checked'),
			selDeviceType = $('#modifyTempDeviceType input:checked').val(),
			curModifySaveUrl = '';
		
		// 选择定制时段，校验是否选择了具体时间 
	   	if( checkedRadio != 1){
	   		//选中的天
	   		$.each($("#modifyTimeSlotSelectDiv .selectRegion_week p"),function(index,ele){
	   			if($(this).hasClass("active")){
	   				week.push($(this).attr("value"));
	   			}
	   		});
	   		//选中的小时点 
	   		$.each($("#modifyTimeSlotSelectDiv .selectRegion_hour p"),function(index,ele){
	   			if($(this).hasClass("active")){
	   				hour.push($(this).html());
	   			}
	   		});
	   	}
		
	   	var selectType = addTemplate.form.selectType;

	 	//选中的设备组
	   	var addChecked = addTemplate.deviceGroupSelection;
		var autoAccessDeviceGroupId = [];
		addChecked.map(function(item,index){
			autoAccessDeviceGroupId.push(item.id);
		})  
		autoAccessDeviceGroupId.sort();
		autoAccessDeviceGroupId = autoAccessDeviceGroupId.join(",");

	    //选中的设备
		var selectedEnbsArr = []; 
	    if(addTemplate.form.devices.length > 0){
	    	selectedEnbsArr = addTemplate.form.devices.split(',');
	    } 
		
		
		//选中的指标
		var selectedKpisArr = []; 
		var selectedKpis = $("#modify_temp_kpi_datagrid").pairgrid("getData");
		if(selectedKpis.length>0){
			$.each(selectedKpis,function(index,ele){
				selectedKpisArr.push(ele.kpiId);
			})
		}
		
		//指标选择为空，给出提示
		if(selectedKpisArr.length == 0){
			$("#modify_temp_kpi_datagrid").parent().next().html("<%=rb.getString("QingXuanZeZhiBiao")%>");
			var domOffset = $("#modify_temp_kpi_datagrid").parent().parent().offset().top;
			var scrollTopSet = domOffset - firstOffset + $("#modifyTemplateDiv_body").scrollTop();
		 	$("#modifyTemplateDiv_body").animate({scrollTop:scrollTopSet},200);
			isBlank=false;
		}
		
		/** 
		* 设备选择方式
		* selectType == 1  设备组
		* selectType == 2 设备
		**/
		
		//设备组选择为空，给出提示
		if(selectType == '1' && addTemplate.deviceGroupSelection.length == 0 && !isAllOperator){
			addTemplate.deviceGroupShow = true;
			isBlank=false;
		}else{
			addTemplate.deviceGroupShow = false;
		}
		
		//设备选择为空，给出提示
		if(selectType == '2' && addTemplate.form.devices.length == 0 && !isAllOperator){
			addTemplate.deviceShow = true;
			isBlank=false;
		}else{
			addTemplate.deviceShow = false;
		}
				
		//kpi数量限制个数
		if(selectedKpisArr.length > kpiLimitNum){	
			$(".kpiTableLimitTip").show();
			isBlank = false;
		}else{
			$(".kpiTableLimitTip").hide();
		}
		/** 
		* 周期设定，选择指定时段时，周或小时未选择时，给出提示
		* checkedRadio == 1   所有时段
		* checkedRadio == 2   指定时段
		**/
		if( checkedRadio != 1){
			if(week.length == 0 || hour.length == 0){
				$("#modifyKpiTimeSlotDiv").next().next().html("<%=rb.getString("QingXuanZeShiJian")%>");
			 	var domOffset = $("#modifyKpiTimeSlotDiv").offset().top;
				var scrollTopSet = domOffset - firstOffset + $("#modifyTemplateDiv_body").scrollTop();
			 	$("#modifyTemplateDiv_body").animate({scrollTop:scrollTopSet},200);
				isBlank=false;
			}
		};
		
		//模板名称为空，给出提示
		if(tempName.length == 0){
			$("#modifyTempName_err").show();
		 	$("#modifyTemplateDiv_body").animate({scrollTop:$("#modifyTemplateDiv_body").offset().top-120},200);
			$("#modifyTempName").focus();
			isBlank=false;
		};
		
		if(!isBlank){
			return;
		}
		var params = {
			"tempId" : tempId,
			"tempName" : tempName,
			"description": description,
			"reportPeriod" : reportPeriod,
			"week" : week.join(","),
			"hour" : hour.join(","),
			"relKpi": selectedKpisArr.join(","),
			"timeZone" : timeZone,
			"is_default": isDefault,
			"selDeviceType": selDeviceType
		};
		// 非全量指标模板无设备选择和校验
		var selectedEnbIsSame = false;
		
		if(!isAllOperator) {
			if(selectType == '1') {
				params["autoAccessDeviceGroupId"] = autoAccessDeviceGroupId;
			}else if(selectType == '2') {
				params["relDevice"] = addTemplate.form.devices;
			}
			
			var oldRelDevice = [];
			if(oldTempData.relDevice){
				oldRelDevice = oldTempData.relDevice.split(',');
			}
			
			$.each(oldRelDevice,function(index,item){
				if($.inArray(oldRelDevice[index],selectedEnbsArr) < 0){
					selectedEnbIsSame = true;
					return;
				}
			});
			
			$.each(selectedEnbsArr,function(index,item){
				if($.inArray(selectedEnbsArr[index],oldRelDevice) < 0){
					selectedEnbIsSame = true;
					return;
				}
			});
		}
		
		var selectedKpiIsSame = false;
		var oldRelKpi = [];
		if(oldTempData.relKpi){
			oldRelKpi = oldTempData.relKpi.split(',');
		}
		$.each(oldRelKpi,function(index,item){
			if($.inArray(oldRelKpi[index],selectedKpisArr)<0){
				selectedKpiIsSame = true;
				return;
			}
		});
		
		$.each(selectedKpisArr,function(index,item){
			if($.inArray(selectedKpisArr[index],oldRelKpi)<0){
				selectedKpiIsSame = true;
				return;
			}
		});
		
		if(!$(".modifyTemp").hasClass("forbidden")){
			if(oldTempData.tempName == tempName && oldTempData.description == description && oldTempData.reportPeriod == reportPeriod
					&& oldTempData.week == week.join(",") && oldTempData.hour == hour.join(",")
					&& autoAccessDeviceGroupId == autoSelDeviceGroup
					&& oldTempData.is_default == (isDefault+'')
					&& !selectedEnbIsSame && !selectedKpiIsSame){
				showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>')
				return;
			}
			
			$(".modifyTemp").addClass("forbidden");
			
			if(curAddEnbGnbEgwType == '0'){
				curModifySaveUrl = '${ctx}/pm/template/addOrModifyTemplate.action';
			}else if(curAddEnbGnbEgwType == '1'){
				curModifySaveUrl = '${ctx}/gnb/pm/template/addOrModifyTemplate.action';
			}else{
				curModifySaveUrl = '${ctx}/egw/pm/template/addOrModifyTemplate.action';
			}
			
			$.post(curModifySaveUrl, params, function(data) {
				if (data["success"]) {
					$("#kpiTempList").datagrid("reload");
					$("#kpiTemplateDatagrid").datagrid("reload");
					showMsg('success_msg','<%=rb.getString("ChengGong")%>')
					closeModifyTemplateDiv();
				} else {
					$(".modifyTemp").removeClass("forbidden");
					showMsg('error_msg',data["message"])
				}
			}, "json");
		}
	}
</script>