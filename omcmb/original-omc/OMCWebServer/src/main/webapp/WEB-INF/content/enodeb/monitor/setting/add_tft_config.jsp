<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#TFTAddPage .container{
		padding-top: 50px;
	}
	#TFTAddPage .addTftFormItem{
		display: flex;
		margin-left:80px;
	}
	#TFTAddPage .addTftFormItem .el-form-item{
		width: 50%;
	}
	#TFTAddPage .addTftFootButton{
		height: 60px;
		line-height: 60px;
		padding-left: 80px;
		position: absolute;
		bottom: 0px;
		left: 0px;
		right: 0px;
		background-color: #FFFFFF;
		border: 1px solid #E9E9E9;
	}
</style>
<div class="pageDefault" id='TFTAddPage'>
	<div class="container">
        <el-form :model="addTftForm" ref="addTftForm" :rules="rules"  label-position="top" label-width="100px" :hide-required-asterisk='true'>
			<div class="addTftFormItem">
				<el-form-item label="PF ID" prop="PF_ID">
					<el-input :disabled="true" v-model="addTftForm.PF_ID" style="padding-top:5px;" ></el-input>
				</el-form-item>
				<el-form-item  label="APP Name" prop="APP_NAME">
					<el-input  v-model="addTftForm.APP_NAME" style="padding-top:5px;" placeholder="<%=rb.getString("ZiFuChang")%><%=rb.getString("MaoHao")%>1-64"></el-input>
				</el-form-item>
			</div>
			<div class="addTftFormItem">
				<el-form-item label="Protocol" prop="PROTOCOL">
					<el-select v-model="addTftForm.PROTOCOL" style="padding-top:5px;" @change="protocolChange">
						<el-option v-for="item in protocolOptions" :key="item.value" :label="item.text" :value="item.value"></el-option>
					</el-select>
				</el-form-item>
				<el-form-item  label="IP_MASK" prop="IP_MASK">
					<el-input  v-model="addTftForm.IP_MASK" style="padding-top:5px;" placeholder="<%=rb.getString("IPMaskGeShiTiShi")%>"></el-input>
				</el-form-item>
			</div>
			<div class="addTftFormItem">
				<el-form-item  label="PORT" prop="PORT">
					<el-input  v-model="addTftForm.PORT" :disabled="portDisabledTag" style="padding-top:5px;"></el-input>
				</el-form-item>
			</div>
		</el-form>
		<div class="addTftFootButton">
			<el-button @click='addTftSubmit' type="primary"><%=rb.getString("QueDing")%></el-button>
			<el-button @click='closeAdd'><%=rb.getString("QuXiao")%></el-button>
		</div>
    </div>
</div>
<script type="text/javascript">
var tb = $('#17EF1A3DD71826B1E0DCED9FDDA82005');
new Vue({
	el:'#TFTAddPage',
	data(){
		var vm = this;
		var validateAPPName = (rule,value,callback) => {
			var reg = /^[\w+$]{1,64}$/;
			if(value === ''){
				callback(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXian")%><%=rb.getString("ZiFuChang")%><%=rb.getString("MaoHao")%>1-64'))
			}else if(reg.test(value)){
				if(vm.appNameList.includes(value) == true){
					callback(new Error('<%=rb.getString("MingChengChongFu")%>'))
				}else{
					callback();
				}
			}else{
				callback(new Error('<%=rb.getString("ZiMuShuZiXiaHuaXian")%><%=rb.getString("ZiFuChang")%><%=rb.getString("MaoHao")%>1-64'))
			}
		};
		var validateIPMask = (rule,value,callback) => {
			var regstr = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\/([0-9]|[12][0-9]|3[012])$/,
				reg = new RegExp(regstr);
			if(value == '' || value == null){
				callback(new Error('<%=rb.getString("IPMaskGeShiTiShi")%>'))
			}else if(reg.test(value)){
				callback();
			}else{
				callback(new Error('<%=rb.getString("IPMaskGeShiTiShi")%>'));
			}
		};
		var validatePort = (rule,value,callback) => {
			if(vm.addTftForm.PROTOCOL == '4'){
				callback();
			}else{
				if(value === ''){
					callback(new Error('<%=rb.getString("BiTian")%>'))
				}else if(vm.isNumeric(value)&& parseInt(value)>=0 && parseInt(value)<=65535){
					callback();
				}else{
					callback(new Error('<%=rb.getString("LGWPortTiShi")%>'))
				}
			}
				
		};
		return {
           	addTftForm:{
                PF_ID:'',
				INDEX:'',
                APP_NAME:'',
                PROTOCOL:'4',
                IP_MASK:'',
                PORT:'',
           	},
			TFTList:[],
			QoSList:[],
			portDisabledTag:true,
			protocolOptions:[
				{value:'4',text:'IP'},
				{value:'6',text:'TCP'},
				{value:'17',text:'UDP'},
			],
			rules:{
				IP_MASK:[
					{validator:validateIPMask,trigger:'blur'}
				],
				PORT:[
					{validator:validatePort,trigger:'blur'}
				],
				APP_NAME:[
					{validator:validateAPPName,trigger:'blur'}
				],
			},
			idList:[],
			indexList:[],
			appNameList:[],
		}
	},
	methods:{
		init(opts,row,type){
			var vm = this;
			vm.rowData = row;
			vm.operType = type;
			vm.getTFTAndQoSTableList();
			Render.validResult.reboot = false;
		},
		protocolChange(val){
			var vm = this;
			if(val == '4'){
				vm.portDisabledTag = true;
				vm.addTftForm.PORT = '';
			}else{
				vm.portDisabledTag = false;
			}
		},
		// 新增提交
		addTftSubmit(){
			var vm = this;
			vm.$refs["addTftForm"].validate( valid => {
				if(valid){
					var parmas ={
							"BC3DC315DF2D8654F8995B0D8696C800":vm.addTftForm.PF_ID,
							"7A8D21C159139F517BB9B592CF8E9ACA":vm.addTftForm.PF_ID,
							"A0A25A7E46F515D47C97BA3D733F32B3":vm.addTftForm.APP_NAME,
							"CB674AAEFC770A68A7D3D5FEC007238E":vm.addTftForm.PROTOCOL,
							"827AE949AC32A97C1FA6F73AEA35A6C8":vm.addTftForm.IP_MASK,
							"F1A1CB61FAA12C13DF9E1E24B2B4EAF0":vm.addTftForm.PORT,
						};
					if(vm.operType == 'add'){
						parmas.operateType = 'add';
						parmas["7A8D21C159139F517BB9B592CF8E9ACA"] = vm.addTftForm.PF_ID;
						vm.setValidResult();
					}else{
						parmas["7A8D21C159139F517BB9B592CF8E9ACA"] = vm.rowData["7A8D21C159139F517BB9B592CF8E9ACA"];
						if(vm.rowData.operateType == 'add'){
							parmas.operateType = 'add';
							Render.tableCollector['17EF1A3DD71826B1E0DCED9FDDA82005'].forEach((items,index,array)=>{
								if(items['BC3DC315DF2D8654F8995B0D8696C800'] == vm.rowData['BC3DC315DF2D8654F8995B0D8696C800']){
									array.splice(items,1)
								}
							})
							$(tb).datagrid('deleteRow', vm.addTftForm.INDEX);
						}else{
							parmas.operateType = 'edit';
							Render.validResult.reboot = true;
						}
					}
					var opts = $(tb).datagrid('options'),idField = opts.idField;
					if(vm.operType == "add"){
						parmas._edit = true;
						parmas._remove = true;
						$(tb).datagrid('appendRow', parmas);

						var rowList = $(tb).datagrid('getRows');

						if(rowList.length>=32) {
							$(tb).parents('.form-item-wrap:first').find('.group-operations .el-icon-plus').hide();
						}
					}else{
						parmas._edit = true;
						parmas._remove = true;
						if(vm.rowData.operateType == 'add'){
							$(tb).datagrid('appendRow', parmas);
						}else{
							$(tb).datagrid('updateRow', {
								index: vm.addTftForm.INDEX,
								row: parmas
							});
						}
						
					}
					delete parmas._edit;
					delete parmas._remove;
					toTbDataQueue(tb, parmas, idField);
					openPropsPanel(false);
				}else{
					return false
				}
			} );
			
		},
		// 取消新增
		closeAdd(){
			openPropsPanel(false);
		},
		getTFTAndQoSTableList(){
			var vm = this;

			vm.TFTList = $(tb).datagrid('getRows');
			if(vm.operType == 'add'){
				
				if(vm.TFTList.length == 0){
					vm.addTftForm.PF_ID = '1';
				}else{
					vm.TFTList.map((item)=>{
						vm.idList.push(parseInt(item["BC3DC315DF2D8654F8995B0D8696C800"]));
						vm.appNameList.push(item["A0A25A7E46F515D47C97BA3D733F32B3"])
					})
					vm.createPfId(1);
				}
			}else{
				vm.addTftForm.PF_ID = vm.rowData["BC3DC315DF2D8654F8995B0D8696C800"]||'';
				vm.addTftForm.INDEX =  $(tb).datagrid('getRowIndex',vm.rowData);
				vm.addTftForm.APP_NAME = vm.rowData["A0A25A7E46F515D47C97BA3D733F32B3"]||'';
				vm.addTftForm.PROTOCOL = vm.rowData["CB674AAEFC770A68A7D3D5FEC007238E"]||'';
				vm.addTftForm.IP_MASK = vm.rowData["827AE949AC32A97C1FA6F73AEA35A6C8"]||'';
				vm.addTftForm.PORT = vm.rowData["F1A1CB61FAA12C13DF9E1E24B2B4EAF0"]||'';

				if(vm.rowData["CB674AAEFC770A68A7D3D5FEC007238E"] != '4'){
					vm.portDisabledTag = false;
				}else{
					vm.portDisabledTag = true;
				}
			}
			
		},
		createPfId(idVal){
			var vm = this;
			if(vm.idList.includes(idVal) == true){
				idVal += 1 ;
				vm.createPfId(idVal);
			}else{
				vm.addTftForm.PF_ID = idVal + '';
			}

		},
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
		// 查看是否有修改或删除的数据  修改状态
		setValidResult(){
			var vm = this;
			if(Render.tableCollector['17EF1A3DD71826B1E0DCED9FDDA82005']){
				Render.tableCollector['17EF1A3DD71826B1E0DCED9FDDA82005'].map((items,index)=>{
					if(items.operateType !== 'add'){
						Render.validResult.reboot = true;
					}
				})
			}
			if(Render.tableCollector['C5F614F36FAFBAA281777B754312924D']){
				Render.tableCollector['C5F614F36FAFBAA281777B754312924D'].map((items,index)=>{
					if(items.operateType !== 'add'){
						Render.validResult.reboot = true;
					}
				})
			}
		}
	},
	mounted(){
        eventBus.$off('tft-init').$on('tft-init',this.init);
	}
	
})

</script> 
