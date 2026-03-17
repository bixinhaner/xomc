<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#addNrCellPage{
		background-color: #FFFFFF;
	}
	#addNrCellPage .logHeader{
		height: 50px;
	}
	#addNrCellPage .headTitleBox{
		height: 50px;
		width: 100%;
		position: relative;
		border-bottom: 1px solid #E9E9E9;
		display: flex;
		align-items: center;
		font-size: 14px;
		font-weight: bold;
		padding-left: 20px;
	}
	#addNrCellPage .logContent{
		height: calc(100% - 80px) !important;
		padding: 30px 0px 30px 20px;
		overflow: auto;
		
	}
	#addNrCellPage .el-collapse-item__header{
		border-bottom:1px solid #fff;
	}
	#addNrCellPage .el-collapse-item__arrow{
		position:absolute;
		left:20px;
		top:0px;
	}
	#addNrCellPage .el-collapse-item{
		position:relative;
	}
	#addNrCellPage .el-icon-arrow-right{
		font-size:16px;
	}
	#addNrCellPage .el-icon-arrow-right:before{
		content:"\e639";
		color:#BBB;
	}
	#addNrCellPage .is-active.el-icon-arrow-right:before{
		content:"\e638";
		color:#BBB;
	}
	#addNrCellPage .el-collapse-item__arrow.is-active{
		transform:rotate(0deg);
	}
	#addNrCellPage .el-collapse{
		border-top:1px solid #fff;
		border-bottom:1px solid #fff;
		margin-bottom: 80px;
	}
	#addNrCellPage .el-collapse-item__wrap{
		border-bottom:1px solid #fff;
	}
	#addNrCellPage .el-collapse-item__header{
		max-width:500px;
	}
	#addNrCellPage .el-collapse-item__header .el-icon::before{
		font-size: 20px;
	}
	#addNrCellPage .el-collapse-item__content{
		padding-bottom: 0px;
	}
	#addNrCellPage .labelIconCls .el-icon::before{
		color: #666666;
		font-size: 12px;
	}
	#addNrCellPage .el-form-item {
		margin-bottom: 20px;
	}
	#addNrCellPage .plmnItemCls{
		margin-left: 20px;
        margin-bottom: 20px;
	}
	#addNrCellPage .plmnItemHeadCls{
		display: flex;
		align-items: center;
		position:relative;
	}
	#addNrCellPage .plmnItemHeadCls .el-form-item{
		width:40%;
		min-width:500px;
		margin-bottom: 0px;
	}
	#addNrCellPage .plmnItemHeadCls .el-form-item__label, .plmnItemFootCls .el-form-item__label{
		font-size: 12px;
	}
	#addNrCellPage .plmnItemFootCls{
		min-height: 100px;
		width: 800px;
		border:1px solid #E9E9E9;
		margin-left: 80px;
		position: relative;
	}
	#addNrCellPage .itemListBoxCls{
		padding-left: 100px;
	}
	#addNrCellPage .itemCls{
		height: 24px;
		display: inline-block;
		line-height: 24px;
		border: 1px solid #4D84FF;
		box-sizing: border-box;
		padding: 0px 10px;
		margin-right: 10px;
		margin-bottom: 10px;
		overflow: hidden;
	}
	#addNrCellPage .itemCls span:first-child{
		display: inline-block;
		padding-right:10px;
		box-sizing: border-box;
	}
	#addNrCellPage .itemListBoxCls .el-icon-close{
		font-size: unset;
		position: unset;
		top: unset;
		right: unset;
	}
	#addNrCellPage .disabledIconBox .el-icon::before{
		color: #e9e9e9;
	}
	#addNrCellPage .sliceListBoxCls .el-input__suffix{
		height: 26px;
		display: flex;
		align-items: center;
	}
	#addNrCellPage .sliceListBoxCls .el-select>.el-input{
		width: 130px;
	}
	#addNrCellPage .mostNumberCls{
		color: #999999;
		margin-left: 10px;
	}
	#addNrCellPage .errorBoxCls{
		color:red;
		font-size:10px;
	}
	#addNrCellPage .plmnListDelIcon{
		position: absolute;
		left: 900px;
		top:8px;
	}
	#addNrCellPage .plmnListDelIcon .el-icon::before{
		font-size: 20px;
	}
	#addNrCellPage .nguIpLabelCls{
		display:inline-block;
		height:24px;
		width:64px;
		line-height: 24px;
		text-align: center;
		border: 1px solid #E9E9E9;
		border-left: none;
		border-right: none;
		background:#F5F7FA;
		position: relative;
		top:0px;
		left: -4px;
	}
	#addNrCellPage .el-form-item__error{
		padding-top: 0px;
	}
	.el-input-group__append{
		border-radius:0px;
		border-right:none;
	}
	.validate-item .el-input__inner{
		width:150px;
	}
	.validate-item .el-input-group__append{
		border:none;
		background:none;
	}
	.validate-item .el-form-item__error{
		display:none;
	}
	.is-error .el-input-group__append{
		color:#FA5555;
	}
</style>
<div class="panelDefault" id="addNrCellPage" style="overflow:hidden">
	<div class="logHeader">
		<div class="headTitleBox">
			{{headTitle}}
			<div style="position:absolute;right:30px;">
				<span class="el-icon el-icon-circle-close" style="margin-left:10px;" @click="closeClick"></span>
			</div>
		</div>
	</div>
	<div class="logContent">
		<el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="left">
			<el-collapse v-model="nrCellCollapse">
				<el-collapse-item name="nrCellSetting">
					<template slot='title'>
						<p style="display:inline-block;margin-left:40px;">
							<span class="title-icon" style="vertical-align:sub"></span>
							<span style="font-size:14px;font-weight:bold">NR Cell Setting</span>
						</p>
					</template>
					<div class="rightContentCls" >
						<div style="display:flex;flex-wrap: wrap;margin-left: 20px;">
							<el-form-item prop='NRCellIdentity' style="width:45%;min-width:540px;" label-width="140px" class='validate-item'>
								<span slot="label" class="labelIconCls">
									NR Cell Identity
									<!--<el-tooltip placement="bottom">
										<div slot="content">
											123456312123151<br/>
											321354534546
										</div>
										<span class="el-icon-circle-info el-icon"></span>
									</el-tooltip>-->
								</span>
								<el-input v-model.trim='ruleForm.NRCellIdentity' style="width:110px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~68719476735,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NrCellTAC' style="width:45%;min-width:540px;"  label-width="140px" class='validate-item'>
								<span slot="label" class="labelIconCls">
									TAC
									<!--<el-tooltip placement="bottom">
										<div slot="content">
											123456312123151<br/>
											321354534546
										</div>
										<span class="el-icon-circle-info el-icon"></span>
									</el-tooltip>-->
								</span>
								<el-input v-model.trim='ruleForm.NrCellTAC' style="width:110px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~16777215,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='NrCellRanac' style="width:45%;min-width:540px;" label-width="140px" class='validate-item'>
								<span slot="label" class="labelIconCls">
									Ranac
									<!--<el-tooltip placement="bottom">
										<div slot="content">
											123456312123151<br/>
											321354534546
										</div>
										<span class="el-icon-circle-info el-icon"></span>
									</el-tooltip>-->
								</span>
								<el-input v-model.trim='ruleForm.NrCellRanac' style="width:110px;padding-top:5px;">
									<template slot="append"><%=rb.getString("FanWei")%>：0~255,Integer</template>
								</el-input>
							</el-form-item>
						</div>
					</div>
				</el-collapse-item>
				<el-collapse-item name="plmnSetting" v-show="plmnSettingShow">
					<template slot='title'>
						<p style="display:inline-block;margin-left:40px;">
							<span class="title-icon" style="vertical-align:sub"></span>
							<span style="font-size:14px;font-weight:bold">PLMN Setting</span>
							<span style="font-size:12px;color:#999999;margin-right:10px;">（No more than 6）</span>
							<span class="el-icon el-icon-circle-add" @click="plmnAddClick" style="position:relative;top:2px;"></span>
						</p>
					</template>
					<div class="rightContentCls" >
						<div v-for="(item,index) in ruleForm.PlmnList" class="plmnItemCls" v-show="!item.operateType || item.operateType && item.operateType != 'remove'">
							<div class="plmnItemHeadCls">
								<div style="width:80px;">PLMN{{index+1}}</div>
								<div style="display:flex;font-size:12px">
									<el-form-item label="PLMN ID"  label-width="100px" > 
										<el-input v-model.trim='item.PlmnId' style="width:110px;padding-top:5px;" @change="plmnIdChange(index,item.PlmnId)"></el-input>
										<span style="margin-left:5px;color:#909399"><%=rb.getString("FanWei")%>：5~6 Digit,Integer</span>
									</el-form-item>
									<el-form-item label="Primary"  label-width="100px">
										<el-radio-group v-model="item.Primary" style="padding-top:10px;">
											<el-radio label="0">0</el-radio>
											<el-radio label="1">1</el-radio>
										</el-radio-group>
									</el-form-item>
								</div>
								<div class="plmnListDelIcon">
									<span class="el-icon el-icon-circle-close" style="margin-left:5px;" @click="plmnListDel(item)"></span>
								</div>
							</div>
							<p class="errorBoxCls" style="margin-left:180px;">{{item.plmnIdErrorMessage}}</p>
							<div class="plmnItemFootCls" v-show="!item.operateType || item.operateType && item.operateType != 'add'">
								<div style="padding:5px 0px 0px 16px;">
									<span style="font-size:12px;font-weight:bold">Slice List</span>
									<span style="font-size:12px;color:#999999;margin-right:10px;">（No more than 6）</span>
								</div>
								<div class="sliceListBoxCls" style="margin-left:16px;">
									<el-form-item label="SNSSAI"  label-width="100px" style="margin-bottom:0px;">
										<el-input style='width:200px;padding-top:7px;' v-model="item.sliceValue">
										</el-input>
										<span class="nguIpLabelCls">NGU IP</span>
										<el-select v-model='item.NguIp' style="width:130px;position:relative;left:-7px;">
											<el-option v-for="(items,index) in nguIpList" :label='items.name' :value='items.value'></el-option>
										</el-select>
										<span class="el-icon el-icon-plus" @click="addSliceVal(index,item.sliceValue,item.NguIp)" v-show="sliceListAddShow(item.sliceList)"></span>
										<div class="disabledIconBox" style="display:inline-block;">
											<i  class="el-icon el-icon-plus" v-show="!sliceListAddShow(item.sliceList)"></i>
										</div>
										<span class="mostNumberCls"><%=rb.getString("FanWei")%>：0~4294967295,Integer</span>
										<p class="errorBoxCls">{{item.sliceListErrorMessage}}</p>
									</el-form-item>
									<div class="itemListBoxCls">
										<div v-for="(items,sliceIndex) in (item.sliceList)" class="itemCls" v-show="!items.operateType || (items.operateType && items.operateType != 'remove')">
											<span style="border-right:1px solid #4d84ff;">{{items.SNSSAI}}</span>
											<span>{{items.NguIp}}</span>
											<span class="el-icon el-icon-close" style="margin-left:5px;" @click="sliceListDel(index,sliceIndex,items)"></span>
										</div>
									</div>
								</div>
							</div>
						</div>
					</div>
					 <el-form-item prop='PlmnList' style="display:none;" label="" label-width="0px">
						<el-input v-model='ruleForm.PlmnList'></el-input>
					</el-form-item>
					<div style="display:none;">{{is_show}}</div>
				</el-collapse-item>
			</el-collapse>
			<div class="footer">
				<div class="lnkbuttonGroup" style="padding:10px 0 0;">
					<el-button type="primary" @click="addNrCellSubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="closeClick"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
		</el-form>
	</div>
</div>

<script type="text/javascript">
	
	var nrCellVue = new Vue({
		el: '#addNrCellPage',
		data(){
			var vm = this;
			var validateRange = (rule,value,callback)=>{
					var min = rule.min;
					var max = rule.max;
					var mag = rule.mag;
					var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
					if(value == '' || value == undefined || value == null){
						callback(new Error(mag))
					}else{
						if(reg.test(value) && value >= min && value <= max){
							callback();
						}else{
							callback(new Error(mag))
						}
					}
					
				};
			return {
				nrCellCollapse:['nrCellSetting','plmnSetting'],
				optType:'',
				ruleForm:{
					NRCellIdentity:'',
					NrCellTAC:'',
					NrCellRanac:'',
					PlmnList:[
						// {Plmn_idx:'1',PlmnId:'',plmnIdErrorMessage:'',Primary:'0',NguIp:'',sliceList:[{SNSSAI:'1',NguIp:'10.1.1.1'},{SNSSAI:'2',NguIp:'10.1.1.1'},{SNSSAI:'3',NguIp:'10.1.1.1'}],sliceValue:'',sliceListErrorMessage:''},
						// {Plmn_idx:'2',PlmnId:'',plmnIdErrorMessage:'',Primary:'1',NguIp:'',sliceList:[{SNSSAI:'1',NguIp:'10.1.1.1'}],sliceValue:'',sliceListErrorMessage:''}
					]
				},
				nguIpList:[],
				rules:{
					NRCellIdentity:[
						{validator:validateRange,min:0,max:68719476735,mag:'<%=rb.getString("FanWei")%>：0~68719476735,Integer'}
					],
					NrCellTAC:[
						{validator:validateRange,min:0,max:16777215,mag:'<%=rb.getString("FanWei")%>：0~16777215,Integer'}
					],
					NrCellRanac:[
						{validator:validateRange,min:0,max:255,mag:'<%=rb.getString("FanWei")%>：0-255,Integer'}
					],
				},
				castsList:{
					'BaiBNX':{
						'QuickSettings':{
							'7E539755DF6E9ACA483677BC91FF796D':'PlmnList',
							'ACF6ECDC67449102099B1AA8EB63E000':'Plmn_idx',
							'470D69649F06F3C2AF1A625E2FF46746':'PlmnId',
							'453F641BA02ACFBE85A6C3F0A7A20E38':'Primary',
							'71F639F9B02C738DF5EBCEC01836FE50':'SliceList',
							'9D00E825C2900CFB655E2D779BD8376C':'Slice_idx',
							'34B5CFDFDAF062D9D3AFEA640F33C77D':'SNSSAI',
							'96395BB50B1008D2B1070FDC01CF8204':'NguIp'
						},
						'RAN':{
							'284C8D3878599989B93BB52FB3E90B74':'PlmnList',
							'F462EAFD037E6D81DDE717CB39D385E4':'Plmn_idx',
							'072D10BE117B1B36CAF726E51653D7B8':'PlmnId',
							'89F68E2911A8B90F9D9284D8C845BCDC':'Primary',
							'93A8927135E6D9D04A01A26E8A23ADE2':'SliceList',
							'EFE76246574B78BBD27A4FCD8C44D9C9':'Slice_idx',
							'AB483D0C8D3FC4F8978D9B87C5FC0118':'SNSSAI',
							'DDF3CCA873BAC2E94B3C8ADAD7380073':'NguIp'
						},
						
					},
					'BaiBNQ':{
						'QuickSettings':{
							'BD9DBC3DA704A5B8FD894D456A96961E':'PlmnList',
							'FDC50DC1194F473C4EBD38BB6A2C76FC':'Plmn_idx',
							'4F7CCEF2044731557F5B9F568FA4D7B9':'PlmnId',
							'9669A2A5A94A97B78AC014FD15DB9C16':'Primary',
							'0CE37FD8EB99D752FD5E48E5D0A7AC6B':'SliceList',
							'338E7490AC59BBD1DFE3590BCBE4A9E6':'Slice_idx',
							'7BFD701CCE9C0D16F8F7610BA6559AD1':'SNSSAI',
							'A27A21EE511324D3DC749B80D2A3AA93':'NguIp'
						},
						'RAN':{
							'8174D9290BA34529A6318E0D1C210F85':'PlmnList',
							'027B2FAB87743EEB5027F18F2D8F5BBB':'Plmn_idx',
							'8BC8EE6BE1E7300FD24465244818BFEF':'PlmnId',
							'7D1C66F18522332AB474CC847E436128':'Primary',
							'163FDB50C10F987C06E8817D7A76BCDF':'SliceList',
							'6D3BA24BFCCA62D7AEB169993BDF5F3E':'Slice_idx',
							'01D100B1F0895DA62A89B02B81C523DA':'SNSSAI',
							'FF9B9482946B8DA3EF145781ECB09647':'NguIp'
						},
					}
				},
				casts:{},
				PlmnList:[],
				codeList:[],
				is_show:false,
				rowData:{},
				productType:'',
				plmnType:'',
			}
		},
		computed: {
			headTitle() {
				var optType={
						'add':'Add',
						'edit':'Modify'
					};

				return optType[this.optType]+' '+'NR Cell'
			},
			cellIndex(){
				var cellName = gnbQuickSettingPageVue.activeName,
					codes={
						'cell1':'1',
						'cell2':'2',
						'cell3':'3',
						'cell4':'4',
					};
				return codes[cellName]
			},
			errorInfoShow(){
				var PlmnList = this.ruleForm.PlmnList,
					errorList=[];
				PlmnList.map((item)=>{
					if(item.plmnIdErrorMessage != ''){
						errorList.push(item)
					}
				})
				return errorList.length>0 ? true : false;
			},
			sliceListAddShow(){
				return (items)=>{
					if(items){
						var arr = items.filter((item)=>{
							return  !item.operateType || (item.operateType &&item.operateType != 'remove')
						})
						return arr.length<6? true : false;
					}else{
						return true
					}
					
				}
			},
			plmnSettingShow(){
				var rowData = this.rowData;

				return this.optType == 'edit' && (!this.rowData.operateType || (this.rowData.operateType &&this.rowData.operateType != 'add'))
			}
		},
		watch: {
			PlmnList(){
				var data = this.PlmnList;
				if(data != null && data != undefined && data != ''){
					var dataVal = JSON.stringify(data);
					this.ruleForm.PlmnList =JSON.parse(dataVal);
					initForm(this.$refs.ruleForm);
				}
			},
		},
		methods: {
			// 初始化
			init(row,optType,productType,plmnType){
				var vm = this,
					codeList=[];
				
				vm.productType = productType;
				vm.casts = vm.castsList[productType][plmnType];

				Object.keys(vm.casts).forEach(function(key){
					codeList.push(key)
				});
				vm.rowData = row;
				vm.optType = optType;
				vm.codeList = codeList;
				vm.plmnType = plmnType;
				if(optType != 'add'){
					if(plmnType = 'QuickSettings'){
						vm.ruleForm.NRCellIdentity = row.NRCellIdentity;
						vm.ruleForm.NrCellTAC = row.NrCellTAC;
						vm.ruleForm.NrCellRanac = row.NrCellRanac;
						vm.ruleForm.NrCell_idx =  row.NrCell_idx;
					}else{
						vm.ruleForm.NRCellIdentity = row.ranNRCellIdentity;
						vm.ruleForm.NrCellTAC = row.ranNrCellTAC;
						vm.ruleForm.NrCellRanac = row.ranNrCellRanac;
						vm.ruleForm.NrCell_idx =  row.ranNrCell_idx;
					}
					
					if(row.operateType){
						vm.ruleForm.operateType = row.operateType;
					}
					var params={
							smallCellCode: gnbQuickSettingPageVue.smallCellCode
						},
						urls='${ctx}/cell/quicksettings/queryNguIpIndex.action';
					axios.post(urls,stringify(params)).then(res=>{
						var data = res.data;
						vm.nguIpList = data;
					})
					vm.initPlmnList();
					initForm(vm.$refs.ruleForm);
				}
				
			},
			initPlmnList(){
				var vm = this
					params = {
						cellIndex:vm.cellIndex,
						taIndex:vm.ruleForm.NrCell_idx,
						smallCellCode: gnbQuickSettingPageVue.smallCellCode
					},
					urls='${ctx}/cell/quicksettings/getListParamValue.action?parent_id=211112&platform=BaiBNX';

				axios.post(urls,stringify(params)).then(res=>{
					var data = res.data;
					if(data.rows){
						var PlmnList=[];
						data.rows.map(item=>{
							var plmn = {};
            				for(var key in item){
            					plmn[vm.casts[key]] = item[key]
            				}
							plmn.NguIp = vm.nguIpList[0].value;
							plmn.sliceValue = '';
							plmn.sliceListErrorMessage = '';
							PlmnList.push(plmn);
						})
						if(PlmnList.length>0){
							PlmnList.map(item=>{
								var sliceList=[],
									sliceParams = {
										cellIndex: vm.cellIndex,
										taIndex: vm.ruleForm.NrCell_idx,
										plmnIndex: item.Plmn_idx,
										smallCellCode: gnbQuickSettingPageVue.smallCellCode
									},
									sliceUrl = '${ctx}/cell/quicksettings/getListParamValue.action?parent_id=2111121&platform=BaiBNX';
								axios.post(sliceUrl,stringify(sliceParams)).then(res=>{
									var sliceData = res.data;
									if(sliceData.rows){
										sliceData.rows.map(sliceItem=>{
											var slice = {};
											for(var key in sliceItem){
												slice[vm.casts[key]] = sliceItem[key]
											}
											sliceList.push(slice);
										})
										sliceList.map((item)=>{
											item.NguIp = vm.formatterNguIpBecomeNguName(item.NguIp);
										})
										vm.$set(item,'sliceList',sliceList);
									}
								})
							})
						}
						vm.PlmnList = PlmnList;
						initForm(vm.$refs.ruleForm);
					}
				})
			},
			addSliceVal(index,value,NguIp){
				var vm = this,
					NguIp = vm.formatterNguIpBecomeNguName(NguIp);
				if(!value)return
				if(vm.isNumeric(value)&&parseInt(value)>=0 && parseInt(value)<=4294967295) {
					// var result = vm.ruleForm.PlmnList[index].sliceList.some(item=>(item.SNSSAI+item.NguIp) == (value+NguIp));
					// if(result){
					// 	vm.ruleForm.PlmnList[index].sliceListErrorMessage = '<%=rb.getString("YiCunZai")%>';
					// }else{
						
					// }
					if(NguIp == ''){
						vm.ruleForm.PlmnList[index].sliceListErrorMessage = '<%=rb.getString("QingXuanZe")%> NGU IP';
					}else{
						var params={
							SNSSAI:value,
							NguIp:NguIp,
							operateType:'add'
						};
						vm.ruleForm.PlmnList[index].sliceList.push(params);
						vm.ruleForm.PlmnList[index].sliceValue = '';
						vm.ruleForm.PlmnList[index].sliceListErrorMessage = '';
					}
					
				}else {
					vm.ruleForm.PlmnList[index].sliceListErrorMessage = '<%=rb.getString("FanWei")%>：0~4294967295,Integer';
				}
			},
			// slice 删除
			sliceListDel(index,sliceIndex,row){
				var vm = this;
				
				if(row.Slice_idx){
						vm.$set(vm.ruleForm.PlmnList[index].sliceList[sliceIndex],'operateType','remove');
				}else{
					vm.ruleForm.PlmnList[index].sliceList.splice(sliceIndex,1);
				}
				vm.$nextTick(()=>{
					vm.is_show = !vm.is_show;
				});
			},
			// 验证输入的是否是数字
			isNumeric(str) {
				if(str.length==0){
					return false;
				}
				for(var i=0;i<str.length;i++){
					if(str.charAt(i)<"0" || str.charAt(i)>"9"){
						return false;
					}
				}
				return true;  
			},
			// plmn 新增
			plmnAddClick(){
				var vm =this,
					params={
						PlmnId:'',
						Primary:'0',
						sliceList:[],
						plmnIdErrorMessage:'',
						operateType:'add',
						taIndex:vm.ruleForm.NrCell_idx,
					};
				var idList=[];
				vm.ruleForm.PlmnList.map((item)=>{
					idList.push(item.Plmn_idx);
				})
				params.Plmn_idx = vm.createId(1,idList); 
				var delNum = 0;
				vm.ruleForm.PlmnList.map((item)=>{
					if(item.operateType&&item.operateType == 'remove'){
						delNum+=1;
					}
				})
				if((vm.ruleForm.PlmnList.length - delNum) < 6){
					vm.ruleForm.PlmnList.push(params)
				}else{
					vm.$message({
						message: 'No more than 6',
						type:'error'
					});
				}
				event.stopPropagation();
			},
			plmnListDel(row){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
                    var delFlag=false;
                    vm.ruleForm.PlmnList.map(function(item,index){
                        if(item.Plmn_idx == row.Plmn_idx){
                            if(item.operateType && item.operateType == 'add'){
                                delFlag = true;
                            }else{
								var params = item;
								params.operateType = 'remove';
								vm.$set(vm.ruleForm.PlmnList,index,params);
                            }
                            
                        }
                    })
                    if(delFlag){
                        vm.ruleForm.PlmnList = vm.ruleForm.PlmnList.filter((items)=>{
                            return items.Plmn_idx != row.Plmn_idx
                        })
                    }
				}).catch(()=>{
					
				})
			},
			addNrCellSubmit(){
				var vm = this,
					isChanged = isFormChanged(vm.$refs.ruleForm);
				
				if(!isChanged){
					showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
					return;
				}
				if(vm.optType == 'edit'){
					var PlmnList = vm.ruleForm.PlmnList,
						errorList=[];
					PlmnList.map((item,index)=>{
						vm.plmnIdChange(index,item.PlmnId)
					})
					if(vm.errorInfoShow){
						return;
					}
					
				}
				vm.$refs.ruleForm.validate(function(valid){
					if(valid){
						var params;
						if(vm.plmnType == 'QuickSettings'){
							params={
								NRCellIdentity:vm.ruleForm.NRCellIdentity,
								NrCellTAC:vm.ruleForm.NrCellTAC,
								NrCellRanac:vm.ruleForm.NrCellRanac,
							};
						}else{
							params={
								ranNRCellIdentity:vm.ruleForm.NRCellIdentity,
								ranNrCellTAC:vm.ruleForm.NrCellTAC,
								ranNrCellRanac:vm.ruleForm.NrCellRanac,
							};
						}
						if(vm.ruleForm.operateType){
							params.operateType = vm.ruleForm.operateType;
						}
						if(vm.cellIndex == '1'){
							if(vm.optType == 'add'){
								params.operateType = 'add';
								var idList=[];
								if(vm.plmnType == 'QuickSettings'){
									cell1InfoVue.ruleForm.NrCellList.map((item)=>{
										idList.push(item.NrCell_idx);
									})
									params.NrCell_idx = vm.createId(1,idList); 
									cell1InfoVue.ruleForm.NrCellList.push(params);
								}else{
									cell1InfoVue.ruleForm.ranNrCellList.map((item)=>{
										idList.push(item.ranNrCell_idx);
									})
									params.ranNrCell_idx = vm.createId(1,idList); 
									cell1InfoVue.ruleForm.ranNrCellList.push(params);
								}
								//eventBus.$emit('close-sharingSlide');
								gnbQuickSettingPageVue.$refs.sharingSlide.hide();
							}else{
								var idx = '';
								if(params.operateType && params.operateType == 'add'){
									params.operateType = 'add'
								}else{
									params.operateType = 'edit';
								}
								if(vm.plmnType == 'QuickSettings'){
									cell1InfoVue.ruleForm.NrCellList.map((item,index)=>{
										if(item.NrCell_idx == vm.ruleForm.NrCell_idx ){
											idx = index;
										}
									})
									Object.assign(cell1InfoVue.ruleForm.NrCellList[idx],params);
								}else{
									cell1InfoVue.ruleForm.ranNrCellList.map((item,index)=>{
										if(item.ranNrCell_idx == vm.ruleForm.NrCell_idx ){
											idx = index;
										}
									})
									Object.assign(cell1InfoVue.ruleForm.ranNrCellList[idx],params);
								}
								vm.PlmnListAndSliceListSubmit();
							}
						}else if(vm.cellIndex == '2'){
							if(vm.optType == 'add'){
								params.operateType = 'add';
								var idList=[];
								if(vm.plmnType == 'QuickSettings'){
									cell2InfoVue.ruleForm.NrCellList.map((item)=>{
										idList.push(item.NrCell_idx);
									})
									params.NrCell_idx = vm.createId(1,idList); 
									cell2InfoVue.ruleForm.NrCellList.push(params);
								}else{
									cell2InfoVue.ruleForm.ranNrCellList.map((item)=>{
										idList.push(item.ranNrCell_idx);
									})
									params.ranNrCell_idx = vm.createId(1,idList); 
									cell2InfoVue.ruleForm.ranNrCellList.push(params);
								}
								//eventBus.$emit('close-sharingSlide');
								gnbQuickSettingPageVue.$refs.sharingSlide.hide();
							}else{
								var idx = '';
								if(params.operateType && params.operateType == 'add'){
									params.operateType = 'add'
								}else{
									params.operateType = 'edit';
								}
								if(vm.plmnType == 'QuickSettings'){
									cell2InfoVue.ruleForm.NrCellList.map((item,index)=>{
										if(item.NrCell_idx == vm.ruleForm.NrCell_idx ){
											idx = index;
										}
									})
									Object.assign(cell2InfoVue.ruleForm.NrCellList[idx],params);
								}else{
									cell2InfoVue.ruleForm.ranNrCellList.map((item,index)=>{
										if(item.ranNrCell_idx == vm.ruleForm.NrCell_idx ){
											idx = index;
										}
									})
									Object.assign(cell2InfoVue.ruleForm.ranNrCellList[idx],params);
								}
								vm.PlmnListAndSliceListSubmit();
							}
						}else if(vm.cellIndex == '3'){
							if(vm.optType == 'add'){
								params.operateType = 'add';
								var idList=[];
								if(vm.plmnType == 'QuickSettings'){
									cell3InfoVue.ruleForm.NrCellList.map((item)=>{
										idList.push(item.NrCell_idx);
									})
									params.NrCell_idx = vm.createId(1,idList); 
									cell3InfoVue.ruleForm.NrCellList.push(params);
								}else{
									cell3InfoVue.ruleForm.ranNrCellList.map((item)=>{
										idList.push(item.ranNrCell_idx);
									})
									params.ranNrCell_idx = vm.createId(1,idList); 
									cell3InfoVue.ruleForm.ranNrCellList.push(params);
								}
								//eventBus.$emit('close-sharingSlide');
								gnbQuickSettingPageVue.$refs.sharingSlide.hide();
							}else{
								var idx = '';
								if(params.operateType && params.operateType == 'add'){
									params.operateType = 'add'
								}else{
									params.operateType = 'edit';
								}
								if(vm.plmnType == 'QuickSettings'){
									cell3InfoVue.ruleForm.NrCellList.map((item,index)=>{
										if(item.NrCell_idx == vm.ruleForm.NrCell_idx ){
											idx = index;
										}
									})
									Object.assign(cell3InfoVue.ruleForm.NrCellList[idx],params);
								}else{
									cell3InfoVue.ruleForm.ranNrCellList.map((item,index)=>{
										if(item.ranNrCell_idx == vm.ruleForm.NrCell_idx ){
											idx = index;
										}
									})
									Object.assign(cell3InfoVue.ruleForm.ranNrCellList[idx],params);
								}
								vm.PlmnListAndSliceListSubmit();
							}
						}else if(vm.cellIndex == '4'){
							if(vm.optType == 'add'){
								params.operateType = 'add';
								var idList=[];
								if(vm.plmnType == 'QuickSettings'){
									cell4InfoVue.ruleForm.NrCellList.map((item)=>{
										idList.push(item.NrCell_idx);
									})
									params.NrCell_idx = vm.createId(1,idList); 
									cell4InfoVue.ruleForm.NrCellList.push(params);
								}else{
									cell4InfoVue.ruleForm.ranNrCellList.map((item)=>{
										idList.push(item.ranNrCell_idx);
									})
									params.ranNrCell_idx = vm.createId(1,idList); 
									cell4InfoVue.ruleForm.ranNrCellList.push(params);
								}
								//eventBus.$emit('close-sharingSlide');
								gnbQuickSettingPageVue.$refs.sharingSlide.hide();
							}else{
								var idx = '';
								if(params.operateType && params.operateType == 'add'){
									params.operateType = 'add'
								}else{
									params.operateType = 'edit';
								}
								
								if(vm.plmnType == 'QuickSettings'){
									cell4InfoVue.ruleForm.NrCellList.map((item,index)=>{
										if(item.NrCell_idx == vm.ruleForm.NrCell_idx ){
											idx = index;
										}
									})
									Object.assign(cell4InfoVue.ruleForm.NrCellList[idx],params);
								}else{
									cell4InfoVue.ruleForm.ranNrCellList.map((item,index)=>{
										if(item.ranNrCell_idx == vm.ruleForm.NrCell_idx ){
											idx = index;
										}
									})
									Object.assign(cell4InfoVue.ruleForm.ranNrCellList[idx],params);
								}
								vm.PlmnListAndSliceListSubmit();
							}
						}
						
					}
				})
			},
			PlmnListAndSliceListSubmit(){
				var vm=this,
					params={},
					changePlmnList=[],
					changeSliceList=[]
					isSync = false;
				vm.$refs.ruleForm.fields.map(function(field){

					if(Array.isArray(field.fieldValue)){
						var vList = field.fieldValue.map(function(item){return item}),
							oList = (field.reinitialValue||[]).map(function(item){return item}),
							newList = vList.sort(),
							oldList = oList.sort();
						for(i=0;i<oldList.length; i++){
							if(!newList[i].operateType){
								if(newList[i].PlmnId != oldList[i].PlmnId || newList[i].Primary != oldList[i].Primary){
									newList[i].operateType = 'edit'
								}
							}
						}
						
						newList.map((item,index)=>{
							if(item.operateType){
								var plmn={
										PlmnId:item.PlmnId,
										Primary:item.Primary,
										cellIndex:vm.cellIndex,
										taIndex:vm.ruleForm.NrCell_idx,
										operateType:item.operateType
									}
								if(item.Plmn_idx){
									plmn.Plmn_idx = item.Plmn_idx;
								}
								changePlmnList.push(plmn)
								if(item.operateType == 'add'){
									isSync = true;
								}
							}
							if(item.sliceList&&item.sliceList.length>0){
								item.sliceList.map((items,indexs)=>{
									if(items.operateType){
										var slice={
												SNSSAI:items.SNSSAI,
												NguIp:vm.formatterNguNameBecomeNguIp(items.NguIp),
												cellIndex:vm.cellIndex,
												taIndex:vm.ruleForm.NrCell_idx,
												plmnIndex:item.Plmn_idx,
												operateType:items.operateType
											}
										if(items.Slice_idx){
											slice.Slice_idx = items.Slice_idx;
										}
										changeSliceList.push(slice)
										if(item.operateType == 'add'){
											isSync = true;
										}
									}
								})
							}
						})
						if(changePlmnList.length>0){
							var subPlmnList = [];
							changePlmnList.map((items)=>{
								var objs={};
								for(var listVal in items){
									var listKey = vm.getNameByProp(listVal);
									objs[listKey] = items[listVal]
								}
								subPlmnList.push(objs)
							})
							if(vm.productType == 'BaiBNX'){
								if(vm.plmnType == 'QuickSettings'){
									params['7E539755DF6E9ACA483677BC91FF796D'] = subPlmnList;
								}else{
									params['284C8D3878599989B93BB52FB3E90B74'] = subPlmnList;
								}
							}else{
								if(vm.plmnType == 'QuickSettings'){
									params['BD9DBC3DA704A5B8FD894D456A96961E'] = subPlmnList;
								}else{
									params['8174D9290BA34529A6318E0D1C210F85'] = subPlmnList;
								}
							}
						}
						if(changeSliceList.length>0){
							var subSliceList = [];
							changeSliceList.map((items)=>{
								var objs={};
								for(var listVal in items){
									var listKey = vm.getNameByProp(listVal);
									objs[listKey] = items[listVal]
								}
								subSliceList.push(objs)
							})
							if(vm.productType == 'BaiBNX'){
								if(vm.plmnType == 'QuickSettings'){
									params['71F639F9B02C738DF5EBCEC01836FE50'] = subSliceList;
								}else{
									params['93A8927135E6D9D04A01A26E8A23ADE2'] = subPlmnList;
								}
								
							}else{
								if(vm.plmnType == 'QuickSettings'){
									params['0CE37FD8EB99D752FD5E48E5D0A7AC6B'] = subSliceList;
								}else{
									params['163FDB50C10F987C06E8817D7A76BCDF'] = subPlmnList;
								}
								
							}
						}
						
					}
					
				});
				if(changePlmnList.length>0 || changeSliceList.length>0){
					var rowCode = gnbQuickSettingPageVue.smallCellCode,
						url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;
					var confirmStr = '<%=rb.getString("5GPLMNSheZhiTiJiaoTiShi")%>'
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(()=>{
						$('#addNrCellPage').addClass('loading');
						axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
							var data = res.data;
							if(data["success"]){
								if(isSync){
									vm.syncParams();
								}
								vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});
								//eventBus.$emit('close-sharingSlide');
								gnbQuickSettingPageVue.$refs.sharingSlide.hide();
							}else{
								vm.$message.error(data["message"])
							}
							$('#addNrCellPage').removeClass('loading');
						})
					}).catch(()=>{
						
					})
				}else{
					//eventBus.$emit('close-sharingSlide');
					gnbQuickSettingPageVue.$refs.sharingSlide.hide();
				}
				
			},
			// 同步
			syncParams(){
				var vm = this,
					urls='${ctx}/cell/quicksettings/sync.action',
					codes = {
						'BaiBNX':'5E33AA12D23A415C0AC4D32E54F26DD7',
						'BaiBNQ':'B779CD488ABB5033764865947EB012D2',
					},
					params={
						smallCellCode: gnbQuickSettingPageVue.smallCellCode,
						paramId:codes[vm.productType],
						cellIndex:vm.cellIndex
					};
				axios.post(urls,stringify(params)).then(res=>{
					var data = res.data;
				})
			},
			createId(idVal,list){
				var vm = this,
					val = idVal + '';
				if(list.includes(val) == true){
					idVal += 1 ;
					return vm.createId(idVal,list);
				}else{
					return  idVal + '';
				}
			},
			closeClick(){
				var vm = this;

				if(vm.optType == 'edit'){
					if(isFormChanged(this.$refs.ruleForm)){
						vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
							customClass:'warningConfirm',
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancelButtonText:'<%=rb.getString("QuXiao")%>',
							type:'warning',
							closeOnClickModal:false
						}).then(() => {
							//eventBus.$emit('close-sharingSlide');
							gnbQuickSettingPageVue.$refs.sharingSlide.hide();
						}).catch(() => {
							
						})
					}else{
						//eventBus.$emit('close-sharingSlide');
						gnbQuickSettingPageVue.$refs.sharingSlide.hide();
					}
				}else{
					//eventBus.$emit('close-sharingSlide');
					gnbQuickSettingPageVue.$refs.sharingSlide.hide();
				}
			},
			// NguIp显示格式化
			formatterNguIpBecomeNguName(val){
				var vm = this,
					NguIpName = '';

				vm.nguIpList.map((item)=>{
					if(item.value == val) {
						NguIpName = item.name
					} 
				})
				return NguIpName
			},
			// NguIp传递格式化
			formatterNguNameBecomeNguIp(name){
				var vm = this,
					NguIp = '';

				vm.nguIpList.map((item)=>{
					if(item.name == name) {
						NguIp = item.value
					} 
				})
				return NguIp
			},
			plmnIdChange(index,val){
				var vm = this,
					reg = /^[0-9]{5,6}$/
											
				if(val ==''){
					vm.ruleForm.PlmnList[index].plmnIdErrorMessage ='Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>';
				}else{
					if(reg.test(val)){
						vm.ruleForm.PlmnList[index].plmnIdErrorMessage = '';
					}else{
						vm.ruleForm.PlmnList[index].plmnIdErrorMessage ='Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>';
					}
				} 
			},
			getNameByProp(prop) {
				var vm = this,
					reg = /^\w*\.\d*\.\w*$/
					key = prop;
				
				if(reg.test(prop)) {
					var mReg = /\.(\d*)\./,
						sufReg = /\.(\w*)$/,
						idx = prop.match(mReg)[1],
						sufStr = prop.match(sufReg)[1];

					vm.codeList.map(function(name){
						var index = vm.indexs[name];
						if(vm.casts[name] == sufStr && index == idx) {
							key = name;
						}
					});
				}else {
					vm.codeList.map(function(name){
						if(vm.casts[name] == prop) {
							key = name;
						}
					});
				}

				return key;
			},
		},
		created(){},
		mounted(){
			eventBus.$off('addNrCell-init').$on('addNrCell-init',this.init);
		}
	});
</script>