<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#kpiAlarmOperPage .el-pairgrid .pairgrid-left{
		border-left:none;
	}
	#kpiAlarmOperPage .selectListCont{
		padding-top:24px;
	}
	#kpiAlarmOperPage .selectListCont .deviceSelectType{
		margin-bottom:16px;
	}
	#kpiAlarmOperPage .productTypeWarp .el-form-item__label{
		line-height: 26px;
		padding: 0 20px;
	}
	#kpiAlarmOperPage .queryGroup{
		height: 26px;
	}
	#kpiAlarmOperPage .pairgrid-query .el-input__inner{
		height: 24px;
		line-height: 24px;
	}
	#kpiAlarmOperPage .kpiSettingSty .el-form-item__label{
		display: none;
	}
	#kpiAlarmOperPage .kpiSettingHeader {
		display: flex;
		border-bottom: 1px solid #D5DCEC;
		background: #F9F9F9;
		height: 30px;
		line-height: 30px;
	}
	#kpiAlarmOperPage .kpiSettingSty .el-select .el-input__suffix {
		right: 2px;
	}
	#kpiAlarmOperPage .kpiListSelection .el-input,
	#kpiAlarmOperPage .kpiListSelection .el-input__inner {
		width: 270px; 
	}
	#kpiAlarmOperPage .resultItem .el-input,
	#kpiAlarmOperPage .resultItem .el-input__inner {
		width: 154px; 
	}
	#kpiAlarmOperPage .compareItem .el-input {
		width: 76px;
	}
	
	
	#kpiAlarmOperPage .thresholdItem .el-input-group__append, 
	#kpiAlarmOperPage .thresholdItem .el-input-group__prepend {
		padding: 0;
		width: 54px;
		text-align: center;
	}
	#kpiAlarmOperPage .limitBox .el-input-group__append,
	#kpiAlarmOperPage .limitBox .el-input-group__prepend {
		padding: 0;
		width: 40px;
		text-align: center;
	}
	#kpiAlarmOperPage .resultItemSty {
		padding-left: 26px;
	}
	#kpiAlarmOperPage .el-select .el-input.is-disabled .el-input__inner,
	#kpiAlarmOperPage .el-select .el-input.is-focus .el-input__inner{
		height: 26px !important;
	}
</style>
<!--新增  KPI 告警模板页面  -->
<div id="kpiAlarmOperPage" class="flex-ctn">
	
	<el-form ref="addform" :model="form" :rules="formRules" label-position="top" style="flex: auto; overflow: auto;">
		<div class="splitGroup">
			<div class="splitGroup_title"><%=rb.getString("JiBenXinXi")%></div>
	  		<div class="splitGroup_body">			
				<el-form-item prop="tempName" label="<%=rb.getString("MingCheng")%>">
					<el-input v-model="form.tempName" :disabled="readonly" placeholder="<%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%> 50" style="width:400px;"></el-input>
				</el-form-item>
				
				<el-form-item prop="description" label="<%=rb.getString("MiaoShu")%>">
					<el-input type="textarea" v-model="form.description" :disabled="readonly" placeholder="<%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%> 200" style="width:400px;"></el-input>
				</el-form-item>
				
				<el-form-item prop="status" label="<%=rb.getString("ZhuangTai")%>">
					<div class="el-textarea__inner" style="padding: 10px 20px;width:400px;">
						<el-radio-group v-model="form.status" :disabled="readonly">
							<el-radio label="on"><%=rb.getString("QiYong")%></el-radio>
							<el-radio label="off"><%=rb.getString("JinYong")%></el-radio>
						</el-radio-group>
					</div>
				</el-form-item>		
			</div>    		
	   	</div>
		
		<div class="splitGroup">
			<div class="splitGroup_title"><%=rb.getString("TiaoJianSheZhi")%></div>
	  		<div class="splitGroup_body">
				<!-- 条件设置 -->
				<label class="el-form-item__label"><%=rb.getString("SheBeiXuanZe")%></label>
				<div class="selectListCont">
					<el-form-item prop="selectType" class="deviceSelectType">
						<div>
							<el-radio-group v-model="form.selectType" :disabled="readonly">
								<el-radio label="group"><%=rb.getString("SheBeiZu")%></el-radio>
								<el-radio label="device"><%=rb.getString("KPISheBei")%></el-radio>
							</el-radio-group>
						</div>
					</el-form-item>
					<!-- 设备组列表 -->
					<el-form-item v-show='showDeviceGroup' prop="groups" label="<%=rb.getString("SheBeiZuLieBiao")%> (<%=rb.getString("XinSheBeiZiDongTianJia")%>)">
						<el-ctable :id="'selected_device_list'" ref="groupTable" :row-key="'id'" :default-checked="defaultCheckedGroup" :url="groupUrl" :height="'360px'" 
							pagination="true" :readonly="readonly" :rownumber="true" @selection-change="groupChange" style="border:1px solid #E9E9E9;">
							<el-table-column label='' type="selection"></el-table-column>
							<el-table-column label='<%=rb.getString("SheBeiZuMingCheng")%>' prop="group_name"></el-table-column>
						</el-ctable>
					</el-form-item>
					<!-- 设备列表 -->
					<el-form-item v-show='!showDeviceGroup' prop="devices">
						<el-pairgrid :id="'select_device_list'" v-if="!readonly"
							:rownumber="true" 
							ref="cpairgrid" 								
							:right-url="rightUrl" 
							:left-url="leftUrl" 
							:height="'360px'" 
							:row-key="'serialNumber'" 
							:query-params="queryForm" 
							:title="deviceTitle"
							:messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}" 
							@checked-change='devicesChange'>
							<template slot="prev">
								<el-ctable style="width:300px;" :id="'group_list'" :show-pager="false" ref="group"  :url="groupUrl" :height="'100%'" :show-header="false" :row-key="'id'" @current-change="queryGroupChange" pagination="false">
									<template slot='toolbar'>
										<%=rb.getString("SheBeiZu")%>
									</template>
									<el-table-column label='<%=rb.getString("SheBeiZu")%>' prop="group_name"></el-table-column>
								</el-ctable>
							</template>
							<template slot="left">
								<el-table-column type="selection" width="45"></el-table-column>
								<el-table-column prop="connection_status" width="50">
									<template slot-scope="scope">
										<div :class="{
											'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
											'':scope.row.have_connected==2,
											'conn_exc':scope.row.connection_status=='Exception',
											'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
									</template>
								</el-table-column>
								<el-table-column prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
								<el-table-column prop='hostName' label='<%=rb.getString("HostName")%>'></el-table-column>
								<el-table-column prop='product' label='<%=rb.getString("ChanPinLeiXing")%>'></el-table-column>
							</template>
							<template slot='toolbar'>
								<div style="display: flex;">
									<el-form-item class="productTypeWarp" label="<%=rb.getString("ChanPinLeiXing")%>"  style="display:flex;">
										<el-select v-model="product_type" @change='productChange'>
											<el-option v-for="item in productTypeList" :key="item.value" :label="item.label" :value="item.value"></el-option>
										</el-select>
									</el-form-item>
									<div class="queryGroup">										
										<div class="el-query" style="position: relative;">																					
											<div class="pairgrid-query el-input el-input--small">
												<input id="search_text" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>" class="el-input__inner">
											</div>
											<i class="el-icon el-icon-common-search" style="margin-left: 10px;" @click="queryDevice"></i>
										</div>
										<input type="hidden"/>
									</div>
								</div>
							</template>
							<template slot='right'>
								<el-table-column prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
								<el-table-column prop='hostName' label='<%=rb.getString("HostName")%>'></el-table-column>
							</template>
						</el-pairgrid>
						<!-- 已选设备列表 只读模式展示 -->
						<el-form-item v-if="readonly" >
							<el-ctable :row-key="'serialNumber'"  :url="selectedDeviceURL" :height="'360px'" pagination="true" :rownumber="true" style="border:1px solid #E9E9E9;">
								<el-table-column prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
								<el-table-column prop='hostName' label='<%=rb.getString("HostName")%>'></el-table-column>
								<el-table-column prop='product' label='<%=rb.getString("ChanPinLeiXing")%>'></el-table-column>
							</el-ctable>
						</el-form-item>
					</el-form-item>
				</div>
				<!-- KPI指标设置 -->
				<el-form-item prop="rules" :label="kpiTitle">
					<div class="el-textarea__inner kpiSettingSty" style="padding: 0;">
						<div class='kpiSettingHeader'>
							<span style="width: 44%; min-width: 744px;text-align: center;"><%=rb.getString("ZhiBiaoMingCheng")%></span>
							<span style="width: 30%;" v-if="operationEnable == '1'"><%=rb.getString("CaoZuo")%></span>
						</div>
						<div v-for="(item,index) in form.rules" style="display: flex; padding: 15px 20px 20px;">
							<div style="display: flex; min-width: 700px; width: 42.5%; padding-right: 24px;">
								<el-form-item label="KPI Threshold" class="thresholdItem"
									:key="item.indicator"
									:prop="'rules.'+index+'.threshold'"
									:rules="[
										{validator: function(rule,value,callback){
											if(item.comparison == 'readonly' || item.comparison == 'none'){
												callback();
											}else{
												if(value == ''){
													callback('<%=rb.getString("QingShuRu")%>');
												}else if(isNaN(value)) {
													callback('<%=rb.getString("MenXianFeiFa")%>');
												}else{
													callback();
												}
											}
											
										}
										}
									]">
									<el-input v-model="item.threshold" controls-position="right" :disabled="readonly || item.comparison== 'none'" maxlength="16" style="width:120px">
										<template slot="append">{{item.unit}}</template>
									</el-input>
								</el-form-item>
								<el-form-item label="Compare" class="compareItem" style="padding: 0 10px;"
									:key="item.indicator"
									:prop="'rules.'+index+'.comparison'"
									:rules="[
										{validator: function(rule,value,callback){
											if(value == '' && item.comparison != 'readonly'){
												callback('<%=rb.getString("QingXuanZe")%>');
											}else{
												callback();
											}
										}
										}
									]">
									<el-select v-model="item.comparison" :disabled="readonly">
										<el-option label="<" value=">"></el-option>
										<el-option label="<=" value=">="></el-option>
										<el-option label="==" value="=="></el-option>
										<el-option label="None" value="none" v-if="item.comparison2 != 'none'"></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label="KPI Name" 
									:key="item.indicator"
									:prop="'rules.'+index+'.indicator'"
									:rules="[
										{required: true,message: '<%=rb.getString("QingXuanZe")%>'}
									]">
									<el-select filterable v-model="item.indicator" :disabled="readonly" @change="kpiChange" class="kpiListSelection">
										<el-option v-for="opItem in kpiList" :label="opItem.text +' ( '+ opItem.value +' )'" :value="opItem.value" :flag="(opItem.value == item.indicator)&&(item.unit = opItem.unit_id)"></el-option>
									</el-select>
								</el-form-item>
								
								<el-form-item label="Compare2" class="compareItem" style="padding: 0 10px;"
									:key="item.indicator"
									:prop="'rules.'+index+'.comparison2'"
									:rules="[
										{validator: function(rule,value,callback){
											
											if(value == '' && item.comparison2 != 'readonly'){
												callback('<%=rb.getString("QingXuanZe")%>');
											}else{
												callback();
											}
										}
										}
									]">
									<el-select v-model="item.comparison2" :disabled="readonly || item.comparison == '=='">
										<el-option label="<" value="<"></el-option>
										<el-option label="<=" value="<="></el-option>				
										<el-option label="None" value="none" v-if="item.comparison != 'none'"></el-option>
									</el-select>
								</el-form-item>
								
								<el-form-item label="KPI Threshold2" class="thresholdItem"
									:key="item.indicator"
									:prop="'rules.'+index+'.threshold2'"
									:rules="[
										{validator: function(rule,value,callback){
											<!--第一个对比符号是 == 或 readonly，则不校验此字段-->
											if(item.comparison2 == 'readonly' || item.comparison == '==' || item.comparison2 == 'none'){
												callback();
											}else{
												if(value == ''){
													callback('<%=rb.getString("QingShuRu")%>');
												}else if(isNaN(value)) {
													callback('<%=rb.getString("MenXianFeiFa")%>');
												}else{
													<!--结束的值要大于开始的值-->
													if(Number(value) <= Number(item.threshold)){
														callback('<%=rb.getString("KPIShuRuCuoWu")%>');
													}else{
														callback();
													}
												}
											}
										}
										}
									]">
									<el-input v-model="item.threshold2" controls-position="right" :disabled="readonly || item.comparison2 == 'none' || item.comparison == '=='" maxlength="16" style="width:120px">
										<template slot="append">{{item.unit}}</template>
									</el-input>
								</el-form-item>
							</div>
							<div style="display: flex;  width: 280px;" v-if="operationEnable == '1'">
								<el-form-item label='' class="resultItem" 
									:key="item.indicator"
									:prop="'rules.'+index+'.operation'"
									:rules="[
										{required: true,message: '<%=rb.getString("QingXuanZe")%>'}
									]">
									<el-select v-model="item.operation" :disabled="readonly">
										<el-option label='<%=rb.getString("GaoJingGuanLi")%>' value="Alarm"></el-option>
										<el-option label='<%=rb.getString("KPIDaiKuanXianZhi")%>' value="Bandwidth Limiting"></el-option>
										<el-option label='<%=rb.getString("KPIQuJiHuoXiaoQu")%>' value="Deactive Cell"></el-option>
									</el-select>
								</el-form-item>
								<el-form-item label='' v-if='item.operation == "Bandwidth Limiting"' class="limitBox" style="padding-left: 10px;" 
										:key="item.indicator"
										:prop="'rules.'+index+'.bandwidth'"
										:rules="[
										{required: true,message: '<%=rb.getString("QingShuRu")%>'},
										{validator: function(rule,value,callback){
											if(isNaN(value)) {
												callback('<%=rb.getString("KPIShuRuCuoWu")%>');
											}else{
												callback();
											}
											}
										}
									]">
									<el-input v-model="item.bandwidth" controls-position="right" :disabled="readonly" style="width: 100px;">
										<template slot="append">M</template>
									</el-input>
								</el-form-item> 
								
							</div>
							<el-form-item label="" style="display: flex;width: 30px;padding-top: 3px" v-if="!readonly">
								<i class="el-icon el-icon-plus" v-if="index===0 && isAddShow" style="cursor:pointer;" @click="updateRule(index)"></i>
								<i class="el-icon el-icon-minus" v-if="index>0" style="margin-top:2px;cursor:pointer" @click="updateRule(index)"></i>
							</el-form-item>
						</div>
					</div>
					</template>
					<template slot='right'>
						<el-table-column prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>(<%=rb.getString("HostName")%>)'></el-table-column>
					</template>
				</el-pairgrid>
			</div>
		</div>
	</el-form>
</div>

<script type="text/javascript">
	/**
	*  页面编辑和只读模式通过readonly控制
	*  校验规则也由readonly决定
	**/
	var addAlarmTemplate = new Vue({
		el: '#kpiAlarmOperPage',
		data(){
			var vm = this;
			
			//校验设备组
			var validateGroup = function(rule,value,callback) { 
				if(vm.showDeviceGroup && value.length == 0) {
					callback('<%=rb.getString("QingXuanZeSheBeiZu")%>');
				}else {
					callback();
				}
			},
				
			// 校验设备
			validateDevice = function(rule,value,callback) { 
				if(!vm.showDeviceGroup && value.length == 0) {
					callback('<%=rb.getString("QingXuanZeSheBei")%>');
				}else {
					callback();
				}
			},
			
			// 校验KPI
			validateRules = function(rule,value,callback) {
				if(value.length > vm.maxRules){
					callback('KPI <%=rb.getString("KPIGuiZeXianZhiGeShu")%> ' +  vm.maxRules  + ' <%=rb.getString("KPIGuiZeXianZhiGeShuTiShi")%>');
				}else if(value.length > 0) {// 存在KPI设置
					var keySet = new Set(),
						isPartFill = false;
				
					value.map(function(item){
						//rules: [{indicator: '',comparison:'',threshold:'',unit:'', comparison2:'',threshold2:'',operation:'',bandwidth:''}],
						//operationEnable: '1' 时，操作显示，否则隐藏
						if(vm.operationEnable == '1'){
							//左对比符+ 左阈值; 右对比符+ 右阈值
							if(item.indicator && item.comparison == 'none' && item.threshold == '' && item.comparison2 && item.threshold2 && item.operation){
								keySet.add(item.indicator + '-' + item.comparison2 + '-' + item.threshold2 + '-' + item.operation);
							}else if(item.indicator && item.comparison2 == 'none' && item.threshold2 == '' && item.comparison && item.threshold && item.operation){
								keySet.add(item.indicator + '-' + item.comparison + '-' + item.threshold + '-' + item.operation);
							}else if(item.indicator && item.comparison && item.threshold && item.comparison2 && item.threshold2 && item.operation){
								var key = item.indicator + '-' + item.comparison + '-' + item.threshold + '-' + item.comparison2 + '-' + item.threshold2 + '-' + item.operation;
								keySet.add(key);
							}else{
								isPartFill = true;
							}
						}else{
							//左对比符+ 左阈值; 右对比符+ 右阈值
							if(item.indicator && item.comparison == 'none' && item.threshold == '' && item.comparison2 && item.threshold2){
								keySet.add(item.indicator + '-' + item.comparison2 + '-' + item.threshold2);
							}else if(item.indicator && item.comparison2 == 'none' && item.threshold2 == '' && item.comparison && item.threshold){
								keySet.add(item.indicator + '-' + item.comparison + '-' + item.threshold);
							}else if(item.indicator && item.comparison && item.threshold && item.comparison2 && item.threshold2){
								var key = item.indicator + '-' + item.comparison + '-' + item.threshold + '-' + item.comparison2 + '-' + item.threshold2;
								keySet.add(key);
							}else{
								isPartFill = true;
							}
						}
					});
					
					if(keySet.size < value.length && !isPartFill) {
						callback('KPI <%=rb.getString("YouChongFuShuJu")%>');
					}else {
						callback();
					}
				}else {
					callback('KPI <%=rb.getString("ZhiShaoTianJiaYiGe")%>');
				}
			};
				
			return {
				queryForm: {
					groupId: '',
					searchText: '',
					product_type: '',
                    isUseTemplate: 'true'
				},
				product_type: '',
				// 表单数据
				form: { 
					tempId: '',
					tempName: '',
					status: 'off',
					description: '',
					selectType: 'device',
					groups: '',
					devices: '',
					rules: [{indicator: '',comparison:'',threshold:'',unit:'', comparison2:'',threshold2:'',operation:'',bandwidth:''}],
					timeZone: timeZone
				},
				productTypeList: [],
				curProduct: '',
				// 校验规则
				rules: { 				
					tempName:[
						{required: true,message:'<%=rb.getString("QingShuRuMuBanMingCheng")%>'},
						{type:'string',max: 50,message:'<%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%> 50'}
					],
					description:[
						{type:'string',max: 200,message:'<%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%> 200'}
					],
					groups:[
						{validator: validateGroup} // 校验设备组
					],
					devices:[
						{validator: validateDevice} // 校验设备
					],					
					rules:[
						{validator: validateRules} // 校验KPI
					]
				},
				kpiList: [],
				maxRules: parseInt('${kpiAlarmThresholdMaxnum}'),
				showPairGrid: true,
		    	leftUrl : '${ctx}/pm/template/getEnbListPageData.action',
		    	rightUrl : '',
		    	selectedDeviceURL: '',
				groupUrl: '',
				deviceTitle: ['<%=rb.getString("JiZhanLieBiao")%>','<%=rb.getString("YiXuanZeJiZhan")%>'],
				groupOptions: [],
				versionOptions: [],
				readonly: false,
				defaultCheckedGroup: [],
				curAlarmProductType:'',
				productTypeListAll: '',
				addProductType: '',
				operationEnable: '${kpi_alarm_operation_enable}', // 是否启用告警操作
			
				kpiTitle: ''
			}
		},
		computed: {
			// 选择类型判断是否显示设备组
			showDeviceGroup(){ 
				return this.form.selectType == 'group';
			},
			// kpi 添加按钮可见逻辑
			isAddShow() { 
				return this.form.rules.length < this.maxRules;
			},
			// 只读模式置空校验
			formRules() { 
				return this.readonly? []:this.rules;
			}
		},
		watch: {
			'form.rules': {
				handler: function(val){
					var vm = this;

					vm.$refs.addform.validateField('rules');
					
					if(val.length > 0){
						val.map(function(item){
							//item.comparison == '==': item.comparison2 参数值 应为 none,直接显示 None， threshold2 参数值置空
							if(item.comparison == '=='){
								item.comparison2 = 'none';
								item.threshold2 = '';
							}
							if(item.comparison == 'none'){
								item.threshold = '';
							}
							//item.threshold2 参数值置空	
							if((item.comparison2 == 'none' && item.comparison == '==') || item.comparison2 == 'none'){
								item.threshold2 = '';
							}
						})
					}
				},
				deep: true
			},
			'form.devices': {
				handler: function(val){
					this.$refs.addform.validateField('devices');
				},
				deep: true
			},
			'form.groups': {
				handler: function(val){
					this.$refs.addform.validateField('groups');
				},
				deep: true
			},
			"form.selectType":function(newVal){
				var vm = this;				
				if(newVal == 'group'){
					vm.curAlarmProductType = vm.addProductType;
				}else{
					var rows = vm.$refs.cpairgrid.getData();
					vm.commonProductType(rows);
				}
			},
			curAlarmProductType: function(newVal,oldVal){
				if(newVal !== oldVal){
					this.getKpiList();
				}
			},
		},
		methods: {
			init(id){
				var vm = this;

				vm.kpiTitle = 'KPI <%=rb.getString("KPIGuiZeXianZhiGeShu")%> ' +  vm.maxRules  + ' <%=rb.getString("KPIGuiZeXianZhiGeShuTiShi")%>'; // 'KPI规则限制个数: n 个
				
		    	vm.selectedDeviceURL = '${ctx}/pm/alarm/getKPIAlarmSelDeviceListPageData.action?tempId='+id;
		    	
		    	// 只读模式
		    	if(vm.readonly) { 
		    		vm.groupUrl = '${ctx}/pm/alarm/getKPIAlarmSelDeviceGroupListPageData.action?tempId='+id;
		    	}else{
		    		vm.groupUrl = '${ctx}/system/deviceGroup/getDeviceGroupList.action';
		    	}
		    	//产品类型
				axios.post('${ctx}/cell/cpeinfos/getEnbMonitorProductList.action?isGnb=0&isAll=1').then(function(response){
					var data = response.data;	
					// 动态删除 BTS
					data = data.filter(function(item) {
						return item !== 'BTS';
					});
					
					if(data.length == 0){
						vm.queryForm.product_type = 'no';
						vm.productTypeListAll = 'no';
					}else{						
						//全部的产品类型 # 51163
						vm.addProductType = data.join(',');
						vm.curAlarmProductType = data.join(',');

						var arr = [{label:'<%=rb.getString("QuanBu")%>',value:''}];

						data.map(function(item){
							if (item){
								arr.push({label:item,value:item})
							}
						});

						vm.productTypeList = arr;
					}
				}).catch(function(error){})

		    	//详情
				axios.post('${ctx}/pm/alarm/getKpiAlarmTempInfo.action','tempId='+id).then(function(response){
					var data = response.data;
					
					if(data) {
						// 往设备选择的右表静态载入数据并初始化级联
						vm.$refs.cpairgrid.$refs.rightTb.rows = data.devices || [];
						vm.$refs.cpairgrid.rightLoadSuccess({rows: (data.devices || [])});
						
						Object.assign(vm.form,data);

						vm.defaultCheckedGroup = data.groups;
						vm.form.devices = data.devices.map(function(item){
							return item.serialNumber;
						}).join(',');

						vm.form.rules = data.rules.map(function(item){
							if(item.hasOwnProperty('operation') && item.operation != null && item.operation != undefined && item.operation != ''){
								if(item.operation.indexOf('Bandwidth Limiting') != -1){
									item.bandwidth = item.operation.match(/\d+/g).join('');
									item.operation = 'Bandwidth Limiting';
								}
							}else{
								item.operation = 'Alarm';
							}

							return item;			
						});
						
						vm.commonProductType(data.devices);						
					}

					vm.readonly = '${type}' == 'view';

					if('${type}' == 'edit'){
						setTimeout(function(){	
							vm.getKpiList();
						},100)
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
					vm.curAlarmProductType = newProductArr.join(',');
				}else{
					//已选设备无产品类型时，则将产品类型接口中返回的全部产品类型作为参数传递
					vm.curAlarmProductType = vm.addProductType;
				}								
			},
			
			// KPI Name 下拉菜单渲染
			getKpiList(params){
				var vm = this, params = {};
				
				if(vm.productTypeListAll == 'no'){
					params.product_type = 'no';
				}else{
					params.product_type = vm.curAlarmProductType;
				}

				axios.post('${ctx}/pm/alarm/getIndicatorsList.action',stringify(params)).then(function(response){
					var data = response.data;
					
					if(data){
						vm.kpiList = data;
						data.map(function(opItem){
							vm.form.rules.map(function(item){
								if(item.indicator == opItem.value){
									item.unit = opItem.unit_id;
								}
							})
						});
					} else{
						vm.kpiList = [];
					}
				}).catch(function(error){})
			},
			// 基站表格查询
			queryDevice(){
				var searchTxt = document.querySelector('#search_text').value;
				this.queryForm.searchText = searchTxt;
				//this.$refs.select_device_list.reload(); // 重新加载表格
			},
			productChange(val){
				var vm = this;

				vm.queryForm.product_type = val;			
			},
			// 设备组列表-当选择项发生变化时触发此事件
			groupChange(value) {
				var vm = this;
				
				vm.$nextTick(function(){
					setTimeout(function(){
						var cks = vm.$refs.groupTable.getChecked();
						vm.form.groups = cks.join(',');
					},0)
				})
			},
			 /**
			*  设备列表里的设备组选项变化
			* @param row{object}: 表格行数据
			**/
			queryGroupChange(row){ 
				if(row) {
					this.queryForm.groupId = row.id;
				}
			},
			
			// 设备选择变化时，更新选择设备记录
			devicesChange(value) {
				var vm = this;
					
				vm.$nextTick(function(){
					var rows = vm.$refs.cpairgrid.getData();
					vm.form.devices = rows.map(function(row){ return row.serialNumber ;}).join(',');
					vm.commonProductType(rows);					
				})			
			},
			
			// KPI Name 选择
			kpiChange(value){
				var vm = this,
					row = vm.kpiList.filter(function(item){
						return item.value == value;
					});

				if(row) {
					vm.form.rules.map(function(item){
						if(item.indicator == row[0].value) item.unit = row[0].unit_id; 
					})
				}
			},
			
			// 添加或删除 KPI 规则门限
			updateRule(index){ 
				if(index) { 
					// 非第一条则为点击删除操作
					this.form.rules.splice(index,1);
				}else { 
					// 第一条则为点击添加操作
					this.form.rules.push({indicator: '',comparison:'',threshold:'',unit:'', comparison2:'',threshold2:'',operation:'', bandwidth:''});
				}
			},

			// 触发新增或修改界面的保存操作
			limitSubmit(){
				var vm = this,
					params = {}, 
					rows = vm.$refs.cpairgrid.getData(),
					kpiRules = vm.form.rules;

				params.tempId = vm.form.tempId;
				params.tempName = vm.form.tempName;
				params.status = vm.form.status;
				params.description = vm.form.description;
				params.selectType = vm.form.selectType;
				params.timeZone = vm.form.timeZone;
				params.groups = vm.form.groups;
				//params.devices = vm.form.devices;
				if(rows.length > 0){
					params.devices = rows.map(function(row){ return row.serialNumber;}).join(',');
				}
				//params.rules = vm.form.rules;
				if(kpiRules.length > 0){
					params.rules = kpiRules.map(function(item){
						var obj = {};
						
						obj.indicator = item.indicator;
						obj.comparison = item.comparison;
						obj.threshold = item.threshold;
						obj.unit = item.unit;
						obj.comparison2 = item.comparison2;
						obj.threshold2 = item.threshold2;

						if(vm.operationEnable == '1'){
							if(item.operation == 'Bandwidth Limiting'){
								obj.operation = 'Bandwidth Limiting' + ' ' + item.bandwidth + 'M';
							}else{
								obj.operation = item.operation;
							}
						}
						return obj;
					})
				}
									
				// 检测模板是否达到最大数，可配置最大数为 10
    	    	axios.post('${ctx}/pm/alarm/checkKpiAlarmTempIsExist.action').then(function(response){
					var data = response.data;
					
					if(data) {
						// 未达到个数上限，允许提交
						if(data['success']) {
							var message = '<%=rb.getString("ChengGong")%>';
						
							vm.$refs.addform.validate(function(valid){
								
								if(valid){
									
									axios.post('${ctx}/pm/alarm/addKpiAlarmTempInfo.action', params).then(function(response){
										
										var data = response.data;
										
					    				if(data["success"]){
					    					vm.$message({
					    						message: message,
					    						type: 'success',
					    						onClose: function(){
					    					    	eventBus.$emit('action-ok');
					    						}
					    					})
					    				}else{
					    					vm.$message.error(data["message"])
					    				}
									}).catch(function(error){})
								}
								
							});
						}else {
							vm.$message.error(data["message"])
						}
					}
				}).catch(function(error){})
			}
		},
		mounted(){
			eventBus.$off('action-save').$on('action-save',this.limitSubmit);
			eventBus.$off('action-init').$on('action-init',this.init);
		}
	});
</script>